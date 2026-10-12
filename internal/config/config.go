package config

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

type Env string

const (
	Local Env = "local"
	Cloud Env = "cloud"
)

type LogFormat string

const (
	LogText LogFormat = "text"
	LogJSON LogFormat = "json"
)

type Config struct {
	Env     Env
	Port    string
	PodName string

	LogLevel  slog.Level
	LogFormat LogFormat

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBSSLMode  string
	DBPassword string
	PANSalt    string

	RunMigrations bool

	AWSRegion          string
	DBPasswordSecretID string
	PANSaltSecretID    string
}

// Load reads the settings through getenv. APP_ENV selects where the secrets
// come from: the environment (local) or AWS Secrets Manager (cloud). The
// settings of the other mode are never read.
func Load(ctx context.Context, getenv func(string) string, secrets SecretGetter) (Config, error) {
	cfg := Config{
		Env:       Env(getenv("APP_ENV")),
		Port:      valueOr(getenv("PORT"), "8080"),
		PodName:   getenv("POD_NAME"),
		DBHost:    getenv("DB_HOST"),
		DBPort:    valueOr(getenv("DB_PORT"), "5432"),
		DBName:    getenv("DB_NAME"),
		DBUser:    getenv("DB_USER"),
		DBSSLMode: valueOr(getenv("DB_SSLMODE"), "require"),
	}

	var missing []string
	require := func(name, value string) {
		if value == "" {
			missing = append(missing, name)
		}
	}

	switch cfg.Env {
	case Local:
		cfg.DBPassword = getenv("DB_PASSWORD")
		cfg.PANSalt = getenv("PAN_SALT")
		require("DB_PASSWORD", cfg.DBPassword)
		require("PAN_SALT", cfg.PANSalt)
	case Cloud:
		cfg.AWSRegion = getenv("AWS_REGION")
		cfg.DBPasswordSecretID = getenv("DB_PASSWORD_SECRET_ID")
		cfg.PANSaltSecretID = getenv("PAN_SALT_SECRET_ID")
		require("AWS_REGION", cfg.AWSRegion)
		require("DB_PASSWORD_SECRET_ID", cfg.DBPasswordSecretID)
		require("PAN_SALT_SECRET_ID", cfg.PANSaltSecretID)
	default:
		missing = append(missing, `APP_ENV (must be "local" or "cloud")`)
	}
	require("DB_HOST", cfg.DBHost)
	require("DB_NAME", cfg.DBName)
	require("DB_USER", cfg.DBUser)

	level, ok := logLevels[valueOr(getenv("LOG_LEVEL"), "info")]
	if !ok {
		missing = append(missing, `LOG_LEVEL (must be "debug", "info", "warn" or "error")`)
	}
	cfg.LogLevel = level

	defaultFormat := LogText
	if cfg.Env == Cloud {
		defaultFormat = LogJSON
	}
	cfg.LogFormat = LogFormat(valueOr(getenv("LOG_FORMAT"), string(defaultFormat)))
	if cfg.LogFormat != LogText && cfg.LogFormat != LogJSON {
		missing = append(missing, `LOG_FORMAT (must be "text" or "json")`)
	}

	cfg.RunMigrations = true
	if raw := getenv("RUN_MIGRATIONS"); raw != "" {
		runMigrations, err := strconv.ParseBool(raw)
		if err != nil {
			missing = append(missing, `RUN_MIGRATIONS (must be "true" or "false")`)
		}
		cfg.RunMigrations = runMigrations
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("invalid environment: %s", strings.Join(missing, ", "))
	}

	if cfg.Env == Cloud {
		var err error
		if cfg.DBPassword, err = readSecret(ctx, secrets, cfg.DBPasswordSecretID); err != nil {
			return Config{}, err
		}
		if cfg.PANSalt, err = readSecret(ctx, secrets, cfg.PANSaltSecretID); err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

// readSecret fails closed: an unreadable or blank secret is an error.
func readSecret(ctx context.Context, secrets SecretGetter, secretID string) (string, error) {
	value, err := secrets.Get(ctx, secretID)
	if err != nil {
		return "", fmt.Errorf("read secret %q: %w", secretID, err)
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("secret %q is empty", secretID)
	}
	return value, nil
}

var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

type SecretGetter interface {
	Get(ctx context.Context, secretID string) (string, error)
}
