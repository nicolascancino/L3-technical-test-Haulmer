package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

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

func (r CardRepository) AddCard(ctx context.Context, c *domain.Card) error {
	row, err := toTable(c)
	if err != nil {
		return err
	}
	err = r.db.WithContext(ctx).Create(&row).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return fmt.Errorf("%w: %s", domain.ErrCardAlreadyExists, c.ID())
	}
	return err
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
