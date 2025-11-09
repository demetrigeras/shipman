package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddress string
	DatabaseDSN string
}

func Load() (*Config, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := env("PGHOST", "localhost")
		port := env("PGPORT", "5432")
		user := env("PGUSER", "demetrigeras")
		password := os.Getenv("PGPASSWORD")
		name := env("PGDATABASE", "shipman")
		sslmode := env("PGSSLMODE", "disable")

		if password != "" {
			dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
		} else {
			dsn = fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s", user, host, port, name, sslmode)
		}
	}

	httpAddr := env("HTTP_ADDR", "0.0.0.0:8080")

	return &Config{
		HTTPAddress: httpAddr,
		DatabaseDSN: dsn,
	}, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
