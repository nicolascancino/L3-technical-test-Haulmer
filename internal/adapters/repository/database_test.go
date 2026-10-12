package repository

import (
	"testing"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
)

func TestDatabaseURL_EscapesCredentials(t *testing.T) {
	cfg := config.Config{DBHost: "db", DBPort: "5432", DBName: "cards", DBUser: "app", DBPassword: "p@ss word", DBSSLMode: "require"}

	got := databaseURL(cfg)

	want := "postgres://app:p%40ss%20word@db:5432/cards?TimeZone=UTC&sslmode=require"
	if got != want {
		t.Errorf("databaseURL() = %q, want %q", got, want)
	}
}
