package command

import (
	"context"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

type CreateCard struct {
	PAN      string
	ExpMonth int
	ExpYear  int
}

type CreateCardHandler struct {
	repo  domain.CardRepository
	newID func() domain.CardID
	now   func() time.Time
}

func NewCreateCardHandler(
	repo domain.CardRepository,
	newID func() domain.CardID,
	now func() time.Time,
) CreateCardHandler {
	if repo == nil {
		panic("nil card repository")
	}
	if newID == nil {
		panic("nil card id generator")
	}
	if now == nil {
		panic("nil clock")
	}
	return CreateCardHandler{repo: repo, newID: newID, now: now}
}

func (h CreateCardHandler) Handle(ctx context.Context, cmd CreateCard) (*domain.Card, error) {
	card, err := domain.NewCard(domain.NewCardConfig{
		ID:        h.newID(),
		PAN:       cmd.PAN,
		ExpMonth:  cmd.ExpMonth,
		ExpYear:   cmd.ExpYear,
		CreatedAt: h.now(),
	})
	if err != nil {
		return nil, err
	}
	if err := h.repo.AddCard(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}
