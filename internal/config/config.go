package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTPAddress   string
	DatabaseDSN   string
	JWTSecret     string
	StoragePath   string
	OpenAIAPIKey  string
	AIProvider    string
	AIModel       string
	AIBaseURL     string
	CoinsubKey        string
	CoinsubMerchantID string
	CoinsubSecret     string
	// RocketRamp / Vantack — peer-to-peer crypto payments via embed wallet.
	RocketRampMerchantID string
	RocketRampAPIKey     string
	RocketRampTestMode   bool
	AppURL        string
	Email         EmailConfig
	MarineAPIKey  string
}

type EmailConfig struct {
	SendGridAPIKey string
	TemplateID     string
	FromAddress    string
	FromName       string
}

type yamlConfig struct {
	Server struct {
		HTTPAddr string `yaml:"http_addr"`
	} `yaml:"server"`

	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"name"`
		SSLMode  string `yaml:"sslmode"`
	} `yaml:"database"`

	Auth struct {
		JWTSecret string `yaml:"jwt_secret"`
	} `yaml:"auth"`

	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`

	AI struct {
		OpenAIAPIKey string `yaml:"openai_api_key"`
		Provider     string `yaml:"provider"` // "openai" or "deepseek"
		Model        string `yaml:"model"`
		BaseURL      string `yaml:"base_url"`
	} `yaml:"ai"`

	Coinsub struct {
		APIKey        string `yaml:"api_key"`
		MerchantID    string `yaml:"merchant_id"`
		WebhookSecret string `yaml:"webhook_secret"`
	} `yaml:"coinsub"`

	RocketRamp struct {
		MerchantID string `yaml:"merchant_id"`
		APIKey     string `yaml:"api_key"`
		TestMode   *bool  `yaml:"test_mode"` // pointer so we can distinguish unset
	} `yaml:"rocketramp"`

	Email struct {
		SendGridAPIKey string `yaml:"sendgrid_api_key"`
		TemplateID     string `yaml:"template_id"`
		FromAddress    string `yaml:"from_address"`
		FromName       string `yaml:"from_name"`
	} `yaml:"email"`

	AppURL       string `yaml:"app_url"`
	MarineAPIKey string `yaml:"marine_traffic_api_key"`
}

func Load() (*Config, error) {
	yc := loadYAML()

	// Environment variables always take priority over YAML
	httpAddr := envOr("HTTP_ADDR", yc.Server.HTTPAddr, "0.0.0.0:8080")
	jwtSecret := envOr("JWT_SECRET", yc.Auth.JWTSecret, "shipman-dev-secret-change-in-production")
	storagePath := envOr("STORAGE_PATH", yc.Storage.Path, "./uploads")
	openAIKey := envOr("OPENAI_API_KEY", yc.AI.OpenAIAPIKey, "")
	aiProvider := envOr("AI_PROVIDER", yc.AI.Provider, "openai")
	aiModel := envOr("AI_MODEL", yc.AI.Model, "")
	aiBaseURL := envOr("AI_BASE_URL", yc.AI.BaseURL, "")

	// Set smart defaults per provider
	if aiModel == "" {
		if aiProvider == "deepseek" {
			aiModel = "deepseek-chat"
		} else {
			aiModel = "gpt-4o-mini"
		}
	}
	if aiBaseURL == "" {
		if aiProvider == "deepseek" {
			aiBaseURL = "https://api.deepseek.com"
		} else {
			aiBaseURL = "https://api.openai.com"
		}
	}
	coinsubKey := envOr("COINSUB_API_KEY", yc.Coinsub.APIKey, "")
	coinsubMerchantID := envOr("COINSUB_MERCHANT_ID", yc.Coinsub.MerchantID, "")
	coinsubSecret := envOr("COINSUB_WEBHOOK_SECRET", yc.Coinsub.WebhookSecret, "")

	rocketRampMerchant := envOr("ROCKETRAMP_MERCHANT_ID", yc.RocketRamp.MerchantID, "")
	rocketRampKey := envOr("ROCKETRAMP_API_KEY", yc.RocketRamp.APIKey, "")
	// Default to PROD (app.myrocketramp.com + api.vantack.com). Sandbox is
	// only used when ROCKETRAMP_TEST_MODE is explicitly set to "true"/"1".
	rocketRampTestMode := false
	if v := os.Getenv("ROCKETRAMP_TEST_MODE"); v != "" {
		rocketRampTestMode = v == "true" || v == "1"
	} else if yc.RocketRamp.TestMode != nil {
		rocketRampTestMode = *yc.RocketRamp.TestMode
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		yamlPort := ""
		if yc.Database.Port > 0 {
			yamlPort = fmt.Sprintf("%d", yc.Database.Port)
		}
		host := envOr("PGHOST", yc.Database.Host, "localhost")
		port := envOr("PGPORT", yamlPort, "5432")
		user := envOr("PGUSER", yc.Database.User, "demetrigeras")
		password := envOr("PGPASSWORD", yc.Database.Password, "")
		name := envOr("PGDATABASE", yc.Database.Name, "shipman")
		sslmode := envOr("PGSSLMODE", yc.Database.SSLMode, "disable")

		if password != "" {
			dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
		} else {
			dsn = fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s", user, host, port, name, sslmode)
		}
	}

	appURL := envOr("APP_URL", yc.AppURL, "http://localhost:3000")
	marineAPIKey := envOr("MARINE_TRAFFIC_API_KEY", yc.MarineAPIKey, "")

	return &Config{
		HTTPAddress:   httpAddr,
		DatabaseDSN:   dsn,
		JWTSecret:     jwtSecret,
		StoragePath:   storagePath,
		OpenAIAPIKey:  openAIKey,
		AIProvider:    aiProvider,
		AIModel:       aiModel,
		AIBaseURL:     aiBaseURL,
		CoinsubKey:        coinsubKey,
		CoinsubMerchantID: coinsubMerchantID,
		CoinsubSecret:     coinsubSecret,
		RocketRampMerchantID: rocketRampMerchant,
		RocketRampAPIKey:     rocketRampKey,
		RocketRampTestMode:   rocketRampTestMode,
		AppURL:        appURL,
		MarineAPIKey:  marineAPIKey,
		Email: EmailConfig{
			SendGridAPIKey: envOr("SENDGRID_API_KEY", yc.Email.SendGridAPIKey, ""),
			TemplateID:     envOr("SENDGRID_TEMPLATE_ID", yc.Email.TemplateID, ""),
			FromAddress:    envOr("EMAIL_FROM", yc.Email.FromAddress, "no-reply@shipman.app"),
			FromName:       envOr("EMAIL_FROM_NAME", yc.Email.FromName, "Shipman"),
		},
	}, nil
}

// loadYAML reads config.local.yaml then config.yaml as fallback, silently skipping if neither exists.
func loadYAML() yamlConfig {
	var yc yamlConfig

	candidates := []string{
		"config/config.local.yaml",
		"config/config.yaml",
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if err := yaml.Unmarshal(data, &yc); err == nil {
			return yc
		}
	}

	return yc
}

// envOr returns the env var if set, otherwise yamlValue, otherwise fallback.
func envOr(envKey, yamlValue, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if yamlValue != "" {
		return yamlValue
	}
	return fallback
}
