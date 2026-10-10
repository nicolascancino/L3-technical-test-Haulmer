package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository/tables"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/query"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}

func (r CardRepository) GetCard(ctx context.Context, id domain.CardID) (query.Card, error) {
	row, err := findCardRow(ctx, r.db, id)
	if err != nil {
		return query.Card{}, err
	}
	return toQueryCard(row), nil
}

func (r CardRepository) ListCards(ctx context.Context, limit int, cursor string) (query.CardPage, error) {
	return r.page(ctx, r.db.Model(&tables.CardRow{}), limit, cursor)
}

func (r CardRepository) SearchCards(ctx context.Context, filter query.CardFilter, limit int, cursor string) (query.CardPage, error) {
	q := r.db.Model(&tables.CardRow{}).Where("last4 = ?", filter.Last4)
	if filter.Brand != "" {
		q = q.Where("brand = ?", filter.Brand)
	}
	if filter.CardHolderName != "" {
		q = q.Where("lower(card_holder_name) LIKE lower(?) || '%'", escapeLike(filter.CardHolderName))
	}
	if filter.ExpMonth != 0 {
		q = q.Where("exp_month = ?", filter.ExpMonth)
	}
	if filter.ExpYear != 0 {
		q = q.Where("exp_year = ?", filter.ExpYear)
	}
	return r.page(ctx, q, limit, cursor)
}

// The cursor is the id of the last row of the previous page. Card ids are UUIDv7 (plan D8),
// so ordering by id is chronological and unique.
func parseCursor(token string) (uuid.UUID, error) {
	id, err := uuid.Parse(token)
	if err != nil {
		return uuid.Nil, domain.ErrInvalidCursor
	}
	return id, nil
}

// It fetches one row more than limit: its presence tells there is a next page.
func (r CardRepository) page(ctx context.Context, q *gorm.DB, limit int, token string) (query.CardPage, error) {
	if token != "" {
		after, err := parseCursor(token)
		if err != nil {
			return query.CardPage{}, err
		}
		q = q.Where("id > ?", after)
	}

	var rows []tables.CardRow
	err := q.WithContext(ctx).Order("id").Limit(limit + 1).Find(&rows).Error
	if err != nil {
		return query.CardPage{}, err
	}

	page := query.CardPage{Items: make([]query.Card, 0, len(rows))}
	if len(rows) > limit {
		rows = rows[:limit]
		page.NextCursor = rows[limit-1].ID
	}
	for _, row := range rows {
		page.Items = append(page.Items, toQueryCard(row))
	}
	return page, nil
}
