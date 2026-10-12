package config

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func validEnv() map[string]string {
	return map[string]string{
		"APP_ENV":     "local",
		"DB_HOST":     "localhost",
		"DB_NAME":     "cards",
		"DB_USER":     "cards",
		"DB_PASSWORD": "secret",
		"PAN_SALT":    "salt",
	}
}

func cloudEnv() map[string]string {
	return map[string]string{
		"APP_ENV":               "cloud",
		"DB_HOST":               "db",
		"DB_NAME":               "cards",
		"DB_USER":               "cards",
		"DB_PASSWORD_SECRET_ID": "cards/db",
		"PAN_SALT_SECRET_ID":    "cards/salt",
		"AWS_REGION":            "us-east-1",
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	cfg, err := load(validEnv())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "8080" || cfg.DBPort != "5432" || cfg.DBSSLMode != "require" {
		t.Errorf("defaults = port %q, db port %q, sslmode %q", cfg.Port, cfg.DBPort, cfg.DBSSLMode)
	}
	if cfg.Env != Local || cfg.DBPassword != "secret" || cfg.PANSalt != "salt" {
		t.Errorf("local values not loaded: %+v", cfg)
	}
}

func TestLoad_ReadsOverrides(t *testing.T) {
	values := validEnv()
	values["PORT"] = "9000"
	values["POD_NAME"] = "cards-api-0"
	values["DB_SSLMODE"] = "disable"

	cfg, err := load(values)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "9000" || cfg.PodName != "cards-api-0" || cfg.DBSSLMode != "disable" {
		t.Errorf("overrides not applied: %+v", cfg)
	}
}

func TestLoad_FailsOnMissingValues(t *testing.T) {
	_, err := load(map[string]string{})
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	for _, name := range []string{"APP_ENV", "DB_HOST", "DB_NAME", "DB_USER"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestLoad_RejectsUnknownAppEnv(t *testing.T) {
	for _, value := range []string{"", "prod", "Local"} {
		values := validEnv()
		values["APP_ENV"] = value

		_, err := load(values)
		if err == nil || !strings.Contains(err.Error(), "APP_ENV") {
			t.Errorf("APP_ENV=%q: Load() error = %v, want APP_ENV error", value, err)
		}
	}
}

func TestLoad_LocalRequiresPlainValues(t *testing.T) {
	values := validEnv()
	delete(values, "DB_PASSWORD")
	delete(values, "PAN_SALT")

	_, err := load(values)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	for _, name := range []string{"DB_PASSWORD", "PAN_SALT"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestLoad_LocalIgnoresSecretSettings(t *testing.T) {
	values := validEnv()
	values["DB_PASSWORD_SECRET_ID"] = "cards/db"
	values["PAN_SALT_SECRET_ID"] = "cards/salt"
	values["AWS_REGION"] = "us-east-1"

	cfg, err := load(values)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPasswordSecretID != "" || cfg.PANSaltSecretID != "" || cfg.AWSRegion != "" {
		t.Errorf("secret settings not ignored: %+v", cfg)
	}
}

func TestLoad_CloudReadsSecretSettings(t *testing.T) {
	cfg, err := load(cloudEnv())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != Cloud || cfg.DBPasswordSecretID != "cards/db" || cfg.PANSaltSecretID != "cards/salt" || cfg.AWSRegion != "us-east-1" {
		t.Errorf("secret settings not loaded: %+v", cfg)
	}
}

func TestLoad_CloudRequiresSecretSettings(t *testing.T) {
	values := cloudEnv()
	delete(values, "DB_PASSWORD_SECRET_ID")
	delete(values, "PAN_SALT_SECRET_ID")
	delete(values, "AWS_REGION")

	_, err := load(values)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	for _, name := range []string{"DB_PASSWORD_SECRET_ID", "PAN_SALT_SECRET_ID", "AWS_REGION"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestLoad_CloudIgnoresPlainValues(t *testing.T) {
	values := cloudEnv()
	values["DB_PASSWORD"] = "plain"
	values["PAN_SALT"] = "plain"

	cfg, err := load(values)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPassword != "db-secret" || cfg.PANSalt != "salt-secret" {
		t.Errorf("plain values were read in cloud: %+v", cfg)
	}
}

func TestLoad_LoggingDefaultsFollowAppEnv(t *testing.T) {
	local, err := load(validEnv())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if local.LogLevel != slog.LevelInfo || local.LogFormat != LogText {
		t.Errorf("local logging = %v, %q", local.LogLevel, local.LogFormat)
	}

	cloud, err := load(cloudEnv())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cloud.LogLevel != slog.LevelInfo || cloud.LogFormat != LogJSON {
		t.Errorf("cloud logging = %v, %q", cloud.LogLevel, cloud.LogFormat)
	}
}

func TestLoad_ReadsLoggingOverrides(t *testing.T) {
	values := cloudEnv()
	values["LOG_LEVEL"] = "debug"
	values["LOG_FORMAT"] = "text"

	cfg, err := load(values)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LogLevel != slog.LevelDebug || cfg.LogFormat != LogText {
		t.Errorf("logging = %v, %q", cfg.LogLevel, cfg.LogFormat)
	}
}

func TestLoad_ReportsInvalidLoggingValuesWithOtherErrors(t *testing.T) {
	values := validEnv()
	delete(values, "DB_HOST")
	values["LOG_LEVEL"] = "verbose"
	values["LOG_FORMAT"] = "xml"

	_, err := load(values)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	for _, name := range []string{"DB_HOST", "LOG_LEVEL", "LOG_FORMAT"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestLoad_RunMigrations(t *testing.T) {
	tests := map[string]bool{"": true, "true": true, "false": false}
	for raw, want := range tests {
		values := validEnv()
		values["RUN_MIGRATIONS"] = raw

		cfg, err := load(values)
		if err != nil {
			t.Fatalf("RUN_MIGRATIONS=%q: Load() error = %v", raw, err)
		}
		if cfg.RunMigrations != want {
			t.Errorf("RUN_MIGRATIONS=%q: RunMigrations = %v, want %v", raw, cfg.RunMigrations, want)
		}
	}
}

func TestLoad_RejectsInvalidRunMigrations(t *testing.T) {
	values := validEnv()
	values["RUN_MIGRATIONS"] = "maybe"

	_, err := load(values)

	if err == nil || !strings.Contains(err.Error(), "RUN_MIGRATIONS") {
		t.Fatalf("Load() error = %v, want error naming RUN_MIGRATIONS", err)
	}
}

type fakeSecrets struct {
	values map[string]string
	errs   map[string]error
}

func (f fakeSecrets) Get(_ context.Context, secretID string) (string, error) {
	return f.values[secretID], f.errs[secretID]
}

func cloudSecrets() fakeSecrets {
	return fakeSecrets{values: map[string]string{"cards/db": "db-secret", "cards/salt": "salt-secret"}}
}

func load(values map[string]string) (Config, error) {
	return Load(context.Background(), env(values), cloudSecrets())
}

func TestLoad_CloudReadsSecretValues(t *testing.T) {
	cfg, err := load(cloudEnv())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPassword != "db-secret" || cfg.PANSalt != "salt-secret" {
		t.Errorf("secrets not loaded: password %q, salt %q", cfg.DBPassword, cfg.PANSalt)
	}
}

func TestLoad_CloudFailsClosed(t *testing.T) {
	tests := map[string]struct {
		secrets  fakeSecrets
		secretID string
	}{
		"read error": {
			secrets:  fakeSecrets{values: map[string]string{"cards/db": "pw"}, errs: map[string]error{"cards/salt": errors.New("denied")}},
			secretID: "cards/salt",
		},
		"empty secret": {
			secrets:  fakeSecrets{values: map[string]string{"cards/db": "", "cards/salt": "salt"}},
			secretID: "cards/db",
		},
		"blank secret": {
			secrets:  fakeSecrets{values: map[string]string{"cards/db": "pw", "cards/salt": "  \n"}},
			secretID: "cards/salt",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Load(context.Background(), env(cloudEnv()), tc.secrets)

			if err == nil || !strings.Contains(err.Error(), tc.secretID) {
				t.Fatalf("Load() error = %v, want error naming %q", err, tc.secretID)
			}
		})
	}
}

func TestLoad_LocalDoesNotReadSecrets(t *testing.T) {
	cfg, err := Load(context.Background(), env(validEnv()), nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPassword != "secret" || cfg.PANSalt != "salt" {
		t.Errorf("local values not loaded: %+v", cfg)
	}
}
