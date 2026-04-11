package users

import (
	"database/sql"
	"net/http"
	"strings"

	"shipman/internal/auth"
	"shipman/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	userRepo   *db.UserRepository
	jwtManager *auth.JWTManager
}

func NewHandler(jwtManager *auth.JWTManager) *Handler {
	return &Handler{
		userRepo:   db.NewUserRepository(),
		jwtManager: jwtManager,
	}
}

type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=shipowner charterer broker"`
}

type SigninRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string  `json:"token"`
	User  db.User `json:"user"`
}

func (h *Handler) AddPublicRoutes(r *gin.RouterGroup) {
	r.POST("/signup", h.handleSignup)
	r.POST("/signin", h.handleSignin)
}

func (h *Handler) AddProtectedRoutes(r *gin.RouterGroup) {
	r.GET("/me", h.handleMe)
}

func (h *Handler) handleSignup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existingUser, err := h.userRepo.RetrieveByEmail(c.Request.Context(), req.Email)
	if err == nil && existingUser.ID != uuid.Nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing user"})
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user := &db.User{
		Email:        strings.ToLower(req.Email),
		PasswordHash: hashedPassword,
		FullName:     req.FullName,
		Role:         req.Role,
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	token, err := h.jwtManager.Generate(user.ID, user.Email, user.Role, user.FullName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token: token,
		User:  *user,
	})
}

func (h *Handler) handleSignin(c *gin.Context) {
	var req SigninRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userRepo.RetrieveByEmail(c.Request.Context(), strings.ToLower(req.Email))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	token, err := h.jwtManager.Generate(user.ID, user.Email, user.Role, user.FullName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User:  user,
	})
}

func (h *Handler) handleMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	user, err := h.userRepo.Retrieve(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	c.JSON(http.StatusOK, user)
}
