package service

import (
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
