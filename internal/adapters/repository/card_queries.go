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

// It fetches one row more than limit: its presence tells there is a next page.
func (r CardRepository) page(ctx context.Context, q *gorm.DB, limit int, token string) (query.CardPage, error) {
	if token != "" {
		c, err := decodeCursor(token)
		if err != nil {
			return query.CardPage{}, err
		}
		q = q.Where("(created_at, id) > (?, ?)", c.CreatedAt, c.ID)
	}

	var rows []tables.CardRow
	err := q.WithContext(ctx).Order("created_at, id").Limit(limit + 1).Find(&rows).Error
	if err != nil {
		return query.CardPage{}, err
	}

	page := query.CardPage{Items: make([]query.Card, 0, len(rows))}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		id, err := uuid.Parse(last.ID)
		if err != nil {
			return query.CardPage{}, err
		}
		page.NextCursor = encodeCursor(cursor{CreatedAt: last.CreatedAt, ID: id})
	}
	for _, row := range rows {
		page.Items = append(page.Items, toQueryCard(row))
	}
	return page, nil
}
