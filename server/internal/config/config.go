package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment             string
	HTTPAddr                string
	DBPath                  string
	JWTSecret               string
	JWTTTL                  time.Duration
	AgentRegistrationKey    string
	HeartbeatTimeout        time.Duration
	TaskLease               time.Duration
	TaskMaxAttempts         int
	DefaultMaxConcurrency   int
	AllowedOrigins          []string
	BootstrapAdminUsername  string
	BootstrapAdminPassword  string
	LogLevel                string
	ArticleRetentionDays    int
	LogRetentionDays        int
	ScreenshotRetentionDays int
}

func Load() (Config, error) {
	cfg := Config{
		Environment:             env("APP_ENV", "development"),
		HTTPAddr:                env("HTTP_ADDR", ":8080"),
		DBPath:                  env("DB_PATH", "./data/platform.db"),
		JWTSecret:               env("JWT_SECRET", "dev-only-change-this-jwt-secret"),
		JWTTTL:                  envDuration("JWT_TTL", 24*time.Hour),
		AgentRegistrationKey:    env("AGENT_REGISTRATION_KEY", "dev-agent-registration-key"),
		HeartbeatTimeout:        envDuration("HEARTBEAT_TIMEOUT", 45*time.Second),
		TaskLease:               envDuration("TASK_LEASE", 90*time.Second),
		TaskMaxAttempts:         envInt("TASK_MAX_ATTEMPTS", 3),
		DefaultMaxConcurrency:   envInt("DEFAULT_AGENT_MAX_CONCURRENCY", 1),
		AllowedOrigins:          splitCSV(env("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		BootstrapAdminUsername:  env("BOOTSTRAP_ADMIN_USERNAME", "admin"),
		BootstrapAdminPassword:  env("BOOTSTRAP_ADMIN_PASSWORD", "admin123!"),
		LogLevel:                env("LOG_LEVEL", "info"),
		ArticleRetentionDays:    envInt("ARTICLE_RETENTION_DAYS", 0),
		LogRetentionDays:        envInt("LOG_RETENTION_DAYS", 30),
		ScreenshotRetentionDays: envInt("SCREENSHOT_RETENTION_DAYS", 7),
	}

	if cfg.DBPath == "" {
		return Config{}, errors.New("DB_PATH is required")
	}
	if len(cfg.JWTSecret) < 24 {
		return Config{}, errors.New("JWT_SECRET must contain at least 24 characters")
	}
	if cfg.AgentRegistrationKey == "" {
		return Config{}, errors.New("AGENT_REGISTRATION_KEY is required")
	}
	if cfg.JWTTTL <= 0 || cfg.HeartbeatTimeout <= 0 || cfg.TaskLease <= 0 {
		return Config{}, errors.New("durations must be positive")
	}
	if cfg.TaskMaxAttempts < 1 || cfg.DefaultMaxConcurrency < 1 {
		return Config{}, errors.New("retry and concurrency values must be positive")
	}
	if cfg.ArticleRetentionDays < 0 ||
		cfg.LogRetentionDays < 0 ||
		cfg.ScreenshotRetentionDays < 0 {
		return Config{}, errors.New("retention days must not be negative")
	}
	if cfg.Environment == "production" {
		if cfg.JWTSecret == "dev-only-change-this-jwt-secret" {
			return Config{}, errors.New("JWT_SECRET must be changed in production")
		}
		if cfg.AgentRegistrationKey == "dev-agent-registration-key" {
			return Config{}, errors.New("AGENT_REGISTRATION_KEY must be changed in production")
		}
		if cfg.BootstrapAdminPassword == "admin123!" {
			return Config{}, errors.New("BOOTSTRAP_ADMIN_PASSWORD must be changed in production")
		}
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}

func splitCSV(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
