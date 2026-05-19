package router

import (
	"net/http"
	"strings"
	"time"

	"shipman/internal/auth"
	"shipman/internal/coinsub"
	"shipman/internal/email"
	"shipman/internal/router/groups/deals"
	"shipman/internal/router/groups/documents"
	"shipman/internal/router/groups/marketplace"
	"shipman/internal/router/groups/users"
	"shipman/internal/router/groups/voyages"
	"shipman/internal/storage"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine       *gin.Engine
	jwtManager   *auth.JWTManager
	storage      storage.Storage
	aiProvider   string
	aiAPIKey     string
	aiModel      string
	aiBaseURL    string
	emailSvc     *email.Service
	appURL       string
	marineAPIKey string
	coinsubClient *coinsub.Client
}

func Setup(jwtSecret string, store storage.Storage, aiProvider, aiAPIKey, aiModel, aiBaseURL string, emailCfg email.Config, appURL, marineAPIKey string, coinsubKey, coinsubMerchantID, coinsubSecret string) *gin.Engine {
	r := &Router{
		engine:        gin.New(),
		jwtManager:    auth.NewJWTManager(jwtSecret, 24*time.Hour),
		storage:       store,
		aiProvider:    aiProvider,
		aiAPIKey:      aiAPIKey,
		aiModel:       aiModel,
		aiBaseURL:     aiBaseURL,
		emailSvc:      email.NewService(emailCfg),
		appURL:        appURL,
		marineAPIKey:  marineAPIKey,
		coinsubClient: coinsub.NewClient(coinsubKey, coinsubMerchantID, coinsubSecret),
	}

	r.engine.Use(gin.Logger())
	r.engine.Use(gin.Recovery())

	r.addDefaultRoutes()
	r.registerAPIRoutes()

	return r.engine
}

func (r *Router) addDefaultRoutes() {
	r.engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "shipman-api",
			"status":  "ok",
			"docs":    "This is the Shipman backend API. Visit the frontend at " + r.appURL,
			"health":  "/healthz",
		})
	})

	r.engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.engine.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

func (r *Router) registerAPIRoutes() {
	api := r.engine.Group("/api")
	api.Use(corsMiddleware())

	v1 := api.Group("/v1")
	v1.Use(requestContextMiddleware(), rateLimitMiddleware())

	userHandler := users.NewHandler(r.jwtManager)

	publicUsers := v1.Group("/users")
	userHandler.AddPublicRoutes(publicUsers)

	protectedUsers := v1.Group("/users")
	protectedUsers.Use(r.authMiddleware())
	userHandler.AddProtectedRoutes(protectedUsers)

	docHandler := documents.NewHandler(r.storage, r.aiProvider, r.aiAPIKey, r.aiModel, r.aiBaseURL)
	// Public route: serve PDF for iframe preview (token passed as query param)
	v1.GET("/documents/:id/view", r.tokenFromQueryMiddleware(), docHandler.HandleView)
	docsGroup := v1.Group("/documents")
	docsGroup.Use(r.authMiddleware())
	docHandler.AddRoutes(docsGroup)

	dealHandler := deals.NewHandler(r.emailSvc, r.appURL)
	publicDeals := v1.Group("/deals")
	dealHandler.AddPublicRoutes(publicDeals)

	dealsGroup := v1.Group("/deals")
	dealsGroup.Use(r.authMiddleware())
	dealHandler.AddRoutes(dealsGroup)

	marketplaceHandler := marketplace.NewHandler()
	marketplaceGroup := v1.Group("/marketplace")
	marketplaceGroup.Use(r.authMiddleware())
	marketplaceHandler.AddRoutes(marketplaceGroup)

	voyageHandler := voyages.NewHandler(r.marineAPIKey, r.aiProvider, r.aiAPIKey, r.aiModel, r.aiBaseURL, r.emailSvc, r.appURL)
	publicVoyages := v1.Group("/voyages")
	voyageHandler.AddPublicRoutes(publicVoyages)

	voyagesGroup := v1.Group("/voyages")
	voyagesGroup.Use(r.authMiddleware())
	voyageHandler.AddRoutes(voyagesGroup)

	paymentHandler := voyages.NewPaymentHandler(r.coinsubClient, r.appURL)
	paymentHandler.AddRoutes(voyagesGroup)
	paymentHandler.AddPublicRoutes(v1)
	paymentHandler.AddUserRoutes(protectedUsers)

	adminGroup := v1.Group("/admin")
	adminGroup.Use(r.authMiddleware())
	paymentHandler.AddAdminRoutes(adminGroup)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func (r *Router) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header format"})
			return
		}

		claims, err := r.jwtManager.Verify(parts[1])
		if err != nil {
			if err == auth.ErrExpiredToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has expired"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", claims.Role)
		c.Set("userFullName", claims.FullName)

		c.Next()
	}
}

// tokenFromQueryMiddleware reads the JWT from ?token= query param (for iframe use).
func (r *Router) tokenFromQueryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			// Also accept Authorization header so the route works either way
			authHeader := c.GetHeader("Authorization")
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 {
				token = parts[1]
			}
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := r.jwtManager.Verify(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Next()
	}
}

func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Remaining", "100")
		c.Header("X-RateLimit-Reset", "60")
		c.Header("X-RateLimit-Window", "60")

		c.Next()
	}
}

func requestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			c.Set("requestID", requestID)
		}
		c.Next()
	}
}
