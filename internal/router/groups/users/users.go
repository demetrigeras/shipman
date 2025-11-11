package users

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AddRoutes(group *gin.RouterGroup) {
	group.POST("/signup", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "signup not implemented"})
	})

	group.POST("/signin", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "signin not implemented"})
	})

	group.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "current user lookup not implemented"})
	})
}
