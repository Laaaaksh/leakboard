// Package config loads Leakboard's runtime configuration from environment
// variables. There is no config file: a self-hosted dashboard's operator
// config is small enough that env vars (and a .env file for local dev) are
// the whole story.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds every environment-derived setting Leakboard needs to run.
type Config struct {
	// Addr is the host:port the HTTP server listens on.
	Addr string
	// DatabaseURL is a standard Postgres connection string.
	DatabaseURL string
	// WorkDir is where repo mirrors are cloned to on disk.
	WorkDir string
	// GitleaksPath is the path to (or name of) the gitleaks binary.
	GitleaksPath string
	// ScanInterval is how often the scheduler checks for repos due a scan.
	ScanInterval time.Duration
	// SessionSecret signs session cookies. Required in production.
	SessionSecret string
	// BaseURL is the externally reachable URL of this instance, used to
	// build links in webhook/email alerts.
	BaseURL string
}

// Load reads configuration from the environment, applying sane defaults for
// everything except DatabaseURL and SessionSecret, which have no safe
// default and must be set explicitly.
func Load() (Config, error) {
	cfg := Config{
		Addr:          getEnv("LEAKBOARD_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("LEAKBOARD_DATABASE_URL"),
		WorkDir:       getEnv("LEAKBOARD_WORKDIR", "./data/repos"),
		GitleaksPath:  getEnv("LEAKBOARD_GITLEAKS_PATH", "gitleaks"),
		SessionSecret: os.Getenv("LEAKBOARD_SESSION_SECRET"),
		BaseURL:       getEnv("LEAKBOARD_BASE_URL", "http://localhost:8080"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("LEAKBOARD_DATABASE_URL is required (e.g. postgres://user:pass@localhost:5432/leakboard)")
	}
	if cfg.SessionSecret == "" {
		return Config{}, fmt.Errorf("LEAKBOARD_SESSION_SECRET is required (any random string, e.g. `openssl rand -hex 32`)")
	}

	intervalSeconds, err := strconv.Atoi(getEnv("LEAKBOARD_SCAN_INTERVAL_SECONDS", "300"))
	if err != nil {
		return Config{}, fmt.Errorf("LEAKBOARD_SCAN_INTERVAL_SECONDS: %w", err)
	}
	cfg.ScanInterval = time.Duration(intervalSeconds) * time.Second

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
