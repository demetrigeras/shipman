package router

import (
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
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}

func registerAPIRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	usersGroup := api.Group("/users")
	users.AddRoutes(usersGroup)
}
