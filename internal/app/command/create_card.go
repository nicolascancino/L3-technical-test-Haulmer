package command

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
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
}

func NewCreateCardHandler(repo domain.CardRepository, hasher PANHasher) CreateCardHandler {
	if repo == nil {
		panic("nil card repository")
	}
	if hasher == nil {
		panic("nil pan hasher")
	}
	return CreateCardHandler{repo: repo, hasher: hasher}
}

func (h CreateCardHandler) Handle(ctx context.Context, cmd CreateCard) (*domain.Card, error) {
	panHash, err := h.hasher.Hash(cmd.PAN)
	if err != nil {
		return nil, fmt.Errorf("hash pan: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate card id: %w", err)
	}

	card, err := domain.NewCard(domain.NewCardConfig{
		ID:             domain.CardID(id.String()),
		PAN:            cmd.PAN,
		ExpMonth:       cmd.ExpMonth,
		ExpYear:        cmd.ExpYear,
		CardHolderName: cmd.CardHolderName,
		PANHash:        panHash,
		CreatedAt:      time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	if err := h.repo.AddCard(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}
