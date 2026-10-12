package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository/tables"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/query"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

// It is both the write port and the read model. The connection must be opened
// with gorm.Config{TranslateError: true} so unique violations surface as
// gorm.ErrDuplicatedKey.
type CardRepository struct {
	db *gorm.DB
}

func NewCardRepository(db *gorm.DB) CardRepository {
	return CardRepository{db: db}
}

func (r CardRepository) AddCard(ctx context.Context, key string, c *domain.Card) (*domain.Card, error) {
	table, err := toTable(c)
	if err != nil {
		return nil, err
	}
	table.IdempotencyKey = key

	db := r.db.WithContext(ctx)
	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(&table)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return nil, fmt.Errorf("%w: %s", domain.ErrCardAlreadyExists, c.ID())
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected > 0 {
		return c, nil
	}

	var existing tables.Card
	if err := db.Take(&existing, "idempotency_key = ?", key).Error; err != nil {
		return nil, err
	}
	return toCard(existing)
}

func (r CardRepository) findCard(ctx context.Context, id domain.CardID) (tables.Card, error) {
	uid, err := parseCardID(id)
	if err != nil {
		return tables.Card{}, err
	}
	var row tables.Card
	err = r.db.WithContext(ctx).Where("id = ?", uid).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tables.Card{}, fmt.Errorf("%w: %s", domain.ErrCardNotFound, id)
	}
	return row, err
}

var (
	_ domain.CardRepository = CardRepository{}
	_ query.CardReader      = CardRepository{}
)
