package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Port    string
	PodName string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBSSLMode  string
	DBPassword string

	AWSRegion          string
	DBPasswordSecretID string
	PANSalt            string
	PANSaltSecretID    string
}

// Load reads the settings through getenv. A secret id replaces its plain
// value (DB_PASSWORD, PAN_SALT), which exists for local runs.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Port:               valueOr(getenv("PORT"), "8080"),
		PodName:            getenv("POD_NAME"),
		DBHost:             getenv("DB_HOST"),
		DBPort:             valueOr(getenv("DB_PORT"), "5432"),
		DBName:             getenv("DB_NAME"),
		DBUser:             getenv("DB_USER"),
		DBSSLMode:          valueOr(getenv("DB_SSLMODE"), "require"),
		DBPassword:         getenv("DB_PASSWORD"),
		AWSRegion:          getenv("AWS_REGION"),
		DBPasswordSecretID: getenv("DB_PASSWORD_SECRET_ID"),
		PANSalt:            getenv("PAN_SALT"),
		PANSaltSecretID:    getenv("PAN_SALT_SECRET_ID"),
	}

	var missing []string
	require := func(name, value string) {
		if value == "" {
			missing = append(missing, name)
		}
	}
	require("DB_HOST", cfg.DBHost)
	require("DB_NAME", cfg.DBName)
	require("DB_USER", cfg.DBUser)
	if cfg.DBPasswordSecretID == "" {
		require("DB_PASSWORD", cfg.DBPassword)
	}
	if cfg.PANSaltSecretID == "" {
		require("PAN_SALT", cfg.PANSalt)
	}
	if cfg.DBPasswordSecretID != "" || cfg.PANSaltSecretID != "" {
		require("AWS_REGION", cfg.AWSRegion)
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
