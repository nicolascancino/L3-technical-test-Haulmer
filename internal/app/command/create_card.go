package command

import (
	"context"
	"fmt"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

type PANHasher interface {
	Hash(pan string) (string, error)
}

type CreateCard struct {
	PAN            string
	CardHolderName string
	ExpMonth       int
	ExpYear        int
}

type CreateCardHandler struct {
	repo   domain.CardRepository
	hasher PANHasher
	newID  func() domain.CardID
	now    func() time.Time
}

func NewCreateCardHandler(
	repo domain.CardRepository,
	hasher PANHasher,
	newID func() domain.CardID,
	now func() time.Time,
) CreateCardHandler {
	if repo == nil {
		panic("nil card repository")
	}
	if hasher == nil {
		panic("nil pan hasher")
	}
	if newID == nil {
		panic("nil card id generator")
	}
	if now == nil {
		panic("nil clock")
	}
	return CreateCardHandler{repo: repo, hasher: hasher, newID: newID, now: now}
}

func (h CreateCardHandler) Handle(ctx context.Context, cmd CreateCard) (*domain.Card, error) {
	panHash, err := h.hasher.Hash(cmd.PAN)
	if err != nil {
		return nil, fmt.Errorf("hash pan: %w", err)
	}

	card, err := domain.NewCard(domain.NewCardConfig{
		ID:             h.newID(),
		PAN:            cmd.PAN,
		ExpMonth:       cmd.ExpMonth,
		ExpYear:        cmd.ExpYear,
		CardHolderName: cmd.CardHolderName,
		PANHash:        panHash,
		CreatedAt:      h.now(),
	})
	if err != nil {
		return nil, err
	}
	if err := h.repo.AddCard(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}
