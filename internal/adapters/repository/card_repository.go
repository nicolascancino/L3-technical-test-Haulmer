package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository/tables"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

// The connection must be opened with gorm.Config{TranslateError: true} so unique
// violations surface as gorm.ErrDuplicatedKey.
type CardRepository struct {
	db *gorm.DB
}

func NewCardRepository(db *gorm.DB) CardRepository {
	return CardRepository{db: db}
}

func (r CardRepository) AddCard(ctx context.Context, c *domain.Card) error {
	row, err := toRow(c)
	if err != nil {
		return err
	}
	record := tables.CardRow(row)
	err = r.db.WithContext(ctx).Create(&record).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return fmt.Errorf("%w: %s", domain.ErrCardAlreadyExists, c.ID())
	}
	return err
}

func (r CardRepository) GetCard(ctx context.Context, id domain.CardID) (*domain.Card, error) {
	row, err := findCardRow(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	return cardRow(row).toDomain()
}

func findCardRow(ctx context.Context, db *gorm.DB, id domain.CardID) (tables.CardRow, error) {
	uid, err := parseCardID(id)
	if err != nil {
		return tables.CardRow{}, err
	}
	var row tables.CardRow
	err = db.WithContext(ctx).Where("id = ?", uid).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tables.CardRow{}, fmt.Errorf("%w: %s", domain.ErrCardNotFound, id)
	}
	return row, err
}
