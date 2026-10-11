package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Config holds the API server settings.
type Config struct {
	ListenAddr      string
	DatabaseURL     string
	MaxOpenConns    int
	ShutdownTimeout time.Duration
}

// LoadConfig reads the API server settings from the environment.
func LoadConfig() (Config, error) {
	cfg := Config{
		ListenAddr:      envOr("LISTEN_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		MaxOpenConns:    20,
		ShutdownTimeout: 15 * time.Second,
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if v := os.Getenv("DB_MAX_OPEN_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, err
		}
		cfg.MaxOpenConns = n
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
