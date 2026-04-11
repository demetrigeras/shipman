package main

import (
	"log"

	"shipman/internal/config"
	"shipman/internal/db"
	"shipman/internal/email"
	"shipman/internal/router"
	"shipman/internal/storage"
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
	log.Println("Connected to PostgreSQL")

	db.SetPool(pool)

	store, err := storage.NewLocalStorage(cfg.StoragePath)
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}
	log.Printf("Storage initialized at %s", cfg.StoragePath)

	emailCfg := email.Config{
		SMTPHost:     cfg.Email.SMTPHost,
		SMTPPort:     cfg.Email.SMTPPort,
		SMTPUser:     cfg.Email.SMTPUser,
		SMTPPassword: cfg.Email.SMTPPassword,
		FromAddress:  cfg.Email.FromAddress,
		FromName:     cfg.Email.FromName,
	}

	r := router.Setup(cfg.JWTSecret, store, cfg.OpenAIAPIKey, cfg.AIModel, cfg.AIBaseURL, emailCfg, cfg.AppURL)

	log.Printf("Starting server on %s", cfg.HTTPAddress)
	if err := r.Run(cfg.HTTPAddress); err != nil {
		log.Fatalf("start http server: %v", err)
	}
}
