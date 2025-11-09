package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"shipman/internal/config"
	"shipman/internal/db"
	"shipman/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := db.Open(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	if err := db.Ping(pool); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL ✅")

	db.SetPool(pool)

	router := gin.Default()
	server.RegisterRoutes(router, pool)

	srv := server.New(router, cfg.HTTPAddress)

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	log.Printf("Server listening on %s", cfg.HTTPAddress)

	shutdown(srv)
}

func shutdown(srv *server.Server) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	} else {
		log.Println("Server stopped gracefully")
	}
}
