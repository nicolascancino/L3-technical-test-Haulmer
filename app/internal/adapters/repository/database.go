package repository

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
)

func OpenDB(cfg config.Config, logger *slog.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL(cfg)), &gorm.Config{
		Logger:         NewGormLogger(logger),
		TranslateError: true,
		NowFunc:        func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

func databaseURL(cfg config.Config) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:     net.JoinHostPort(cfg.DBHost, cfg.DBPort),
		Path:     cfg.DBName,
		RawQuery: url.Values{"sslmode": {cfg.DBSSLMode}, "TimeZone": {"UTC"}}.Encode(),
	}
	return u.String()
}
