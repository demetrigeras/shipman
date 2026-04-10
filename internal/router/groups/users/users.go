package users

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AddRoutes(r *gin.RouterGroup) {
	r.POST("/signup", func(c *gin.Context) {
		c.Status(http.StatusNotImplemented)
	})

	r.POST("/signin", func(c *gin.Context) {
		c.Status(http.StatusNotImplemented)
	})

	r.GET("/me", func(c *gin.Context) {
		c.Status(http.StatusNotImplemented)
	})
}
