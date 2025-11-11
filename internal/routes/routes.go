package routes

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	engine *gin.Engine
	http   *http.Server
}

func New(engine *gin.Engine, addr string) *Server {
	return &Server{
		engine: engine,
		http: &http.Server{
			Addr:              addr,
			Handler:           engine,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	r.GET("/healthz", func(c *gin.Context) {
		if err := db.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.GET("/charters", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		})
	}
}
