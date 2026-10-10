package service

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository/tables"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/secrets"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/security"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/command"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/query"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
)

func NewApplication(ctx context.Context, cfg config.Config) (app.Application, error) {
	dbPassword, panSalt := cfg.DBPassword, cfg.PANSalt
	if cfg.Env == config.Cloud {
		awsSecrets, err := secrets.NewAWSClient(ctx, cfg.AWSRegion)
		if err != nil {
			return app.Application{}, err
		}
		if dbPassword, panSalt, err = readCloudSecrets(ctx, awsSecrets, cfg); err != nil {
			return app.Application{}, err
		}
	}

	db, err := gorm.Open(postgres.Open(databaseURL(cfg, dbPassword)), &gorm.Config{TranslateError: true})
	if err != nil {
		return app.Application{}, fmt.Errorf("open database: %w", err)
	}
	if err := tables.Migrate(db); err != nil {
		return app.Application{}, fmt.Errorf("migrate database: %w", err)
	}

	cards := repository.NewCardRepository(db)
	hasher := security.NewHMACPANHasher(panSalt)

	return app.Application{
		Commands: app.Commands{
			CreateCard: command.NewCreateCardHandler(cards, hasher),
		},
		Queries: app.Queries{
			GetCard:     query.NewGetCardHandler(cards),
			ListCards:   query.NewListCardsHandler(cards),
			SearchCards: query.NewSearchCardsHandler(cards),
		},
	}, nil
}

func databaseURL(cfg config.Config, password string) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.DBUser, password),
		Host:     net.JoinHostPort(cfg.DBHost, cfg.DBPort),
		Path:     cfg.DBName,
		RawQuery: url.Values{"sslmode": {cfg.DBSSLMode}}.Encode(),
	}
	return u.String()
}
