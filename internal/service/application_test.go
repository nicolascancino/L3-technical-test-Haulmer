package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
)

func TestDatabaseURL_EscapesCredentials(t *testing.T) {
	cfg := config.Config{DBHost: "db", DBPort: "5432", DBName: "cards", DBUser: "app", DBSSLMode: "require"}

	got := databaseURL(cfg, "p@ss word")

	want := "postgres://app:p%40ss%20word@db:5432/cards?sslmode=require"
	if got != want {
		t.Errorf("databaseURL() = %q, want %q", got, want)
	}
}

type fakeSecrets struct {
	values map[string]string
	errs   map[string]error
}

func (f fakeSecrets) Get(_ context.Context, secretID string) (string, error) {
	return f.values[secretID], f.errs[secretID]
}

func TestReadCloudSecrets_ReturnsBothValues(t *testing.T) {
	cfg := config.Config{DBPasswordSecretID: "cards/db", PANSaltSecretID: "cards/salt"}
	getter := fakeSecrets{values: map[string]string{"cards/db": "pw", "cards/salt": "salt"}}

	password, salt, err := readCloudSecrets(context.Background(), getter, cfg)
	if err != nil {
		t.Fatalf("readCloudSecrets() error = %v", err)
	}
	if password != "pw" || salt != "salt" {
		t.Errorf("got %q, %q", password, salt)
	}
}

func TestReadCloudSecrets_FailsClosed(t *testing.T) {
	cfg := config.Config{DBPasswordSecretID: "cards/db", PANSaltSecretID: "cards/salt"}
	tests := map[string]struct {
		getter   fakeSecrets
		secretID string
	}{
		"read error": {
			getter:   fakeSecrets{values: map[string]string{"cards/db": "pw"}, errs: map[string]error{"cards/salt": errors.New("denied")}},
			secretID: "cards/salt",
		},
		"empty secret": {
			getter:   fakeSecrets{values: map[string]string{"cards/db": "", "cards/salt": "salt"}},
			secretID: "cards/db",
		},
		"blank secret": {
			getter:   fakeSecrets{values: map[string]string{"cards/db": "pw", "cards/salt": "  \n"}},
			secretID: "cards/salt",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, err := readCloudSecrets(context.Background(), tc.getter, cfg)
			if err == nil || !strings.Contains(err.Error(), tc.secretID) {
				t.Fatalf("readCloudSecrets() error = %v, want error naming %q", err, tc.secretID)
			}
		})
	}
}
