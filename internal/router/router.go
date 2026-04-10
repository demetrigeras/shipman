package router

import (
	"net/http"

	"shipman/internal/router/groups/users"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	addDefaultRoutes(r)
	registerAPIRoutes(r)

	return r
}

func addDefaultRoutes(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

func registerAPIRoutes(r *gin.Engine) {
	routes(r)
}

func routes(r *gin.Engine) {
	api := r.Group("/api")
	api.Use(corsMiddleware())

	v1 := api.Group("/v1")
	v1.Use(requestContextMiddleware(), rateLimitMiddleware())

	usersGroup := v1.Group("/users")
	usersGroup.Use(authMiddleware())
	users.AddRoutes(usersGroup)
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

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}

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
