package config

import (
	"strings"
	"testing"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func validEnv() map[string]string {
	return map[string]string{
		"DB_HOST":     "localhost",
		"DB_NAME":     "cards",
		"DB_USER":     "cards",
		"DB_PASSWORD": "secret",
		"PAN_SALT":    "salt",
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	cfg, err := Load(env(validEnv()))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "8080" || cfg.DBPort != "5432" || cfg.DBSSLMode != "require" {
		t.Errorf("defaults = port %q, db port %q, sslmode %q", cfg.Port, cfg.DBPort, cfg.DBSSLMode)
	}
	if cfg.DBPassword != "secret" || cfg.PANSalt != "salt" {
		t.Errorf("fallback values not loaded: %+v", cfg)
	}
}

func TestLoad_ReadsOverrides(t *testing.T) {
	values := validEnv()
	values["PORT"] = "9000"
	values["POD_NAME"] = "cards-api-0"
	values["DB_SSLMODE"] = "disable"

	cfg, err := Load(env(values))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "9000" || cfg.PodName != "cards-api-0" || cfg.DBSSLMode != "disable" {
		t.Errorf("overrides not applied: %+v", cfg)
	}
}

func TestLoad_FailsOnMissingValues(t *testing.T) {
	_, err := Load(env(map[string]string{}))
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	for _, name := range []string{"DB_HOST", "DB_NAME", "DB_USER", "DB_PASSWORD", "PAN_SALT"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestLoad_SecretIDReplacesFallbackValue(t *testing.T) {
	values := validEnv()
	delete(values, "DB_PASSWORD")
	delete(values, "PAN_SALT")
	values["DB_PASSWORD_SECRET_ID"] = "cards/db"
	values["PAN_SALT_SECRET_ID"] = "cards/salt"
	values["AWS_REGION"] = "us-east-1"

	cfg, err := Load(env(values))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPasswordSecretID != "cards/db" || cfg.PANSaltSecretID != "cards/salt" || cfg.AWSRegion != "us-east-1" {
		t.Errorf("secret settings not loaded: %+v", cfg)
	}
}

func TestLoad_SecretIDRequiresRegion(t *testing.T) {
	values := validEnv()
	values["PAN_SALT_SECRET_ID"] = "cards/salt"

	_, err := Load(env(values))
	if err == nil || !strings.Contains(err.Error(), "AWS_REGION") {
		t.Fatalf("Load() error = %v, want AWS_REGION error", err)
	}
}
