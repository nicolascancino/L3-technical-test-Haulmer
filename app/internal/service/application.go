package service

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository/tables"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/security"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/command"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/query"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
)

func NewApplication(cfg config.Config, db *gorm.DB) (app.Application, error) {
	if cfg.RunMigrations {
		if err := tables.Migrate(db); err != nil {
			return app.Application{}, fmt.Errorf("migrate database: %w", err)
		}
	}

	cards := repository.NewCardRepository(db)
	hasher := security.NewHMACPANHasher(cfg.PANSalt)

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
