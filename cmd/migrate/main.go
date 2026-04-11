package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"shipman/internal/config"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("connect to database: %v\n\nCheck your config/config.local.yaml or environment variables.", err)
	}

	// Ensure the shipman schema exists before goose tries to access its version table
	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS shipman`); err != nil {
		log.Fatalf("create schema: %v", err)
	}

	// Keep goose's version table inside the shipman schema
	goose.SetTableName("shipman.goose_db_version")
	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("set dialect: %v", err)
	}

	migrationsDir := "db/migrations"
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "up":
		if err := goose.Up(db, migrationsDir); err != nil {
			log.Fatalf("migrate up: %v", err)
		}
		fmt.Println("Migrations applied successfully.")

	case "down":
		if err := goose.Down(db, migrationsDir); err != nil {
			log.Fatalf("migrate down: %v", err)
		}
		fmt.Println("Last migration rolled back.")

	case "down-to":
		if len(os.Args) < 3 {
			log.Fatal("usage: migrate down-to <version>")
		}
		version, err := goose.EnsureDBVersion(db)
		if err != nil {
			log.Fatalf("get version: %v", err)
		}
		fmt.Printf("Current version: %d\n", version)

	case "reset":
		if err := goose.Reset(db, migrationsDir); err != nil {
			log.Fatalf("migrate reset: %v", err)
		}
		fmt.Println("All migrations rolled back.")

	case "status":
		if err := goose.Status(db, migrationsDir); err != nil {
			log.Fatalf("migrate status: %v", err)
		}

	case "version":
		version, err := goose.GetDBVersion(db)
		if err != nil {
			log.Fatalf("get version: %v", err)
		}
		fmt.Printf("Current migration version: %d\n", version)

	default:
		log.Fatalf("unknown command: %s\n\nAvailable commands: up, down, reset, status, version", command)
	}
}
