package main

import (
	"log"

	"shipman/internal/config"
	"shipman/internal/db"
	"shipman/internal/router"
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

	r := router.Setup()

	if err := r.Run(cfg.HTTPAddress); err != nil {
		log.Fatalf("start http server: %v", err)
	}
}
