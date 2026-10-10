package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/command"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

type fakeRepo struct {
	added  []*domain.Card
	addErr error
}

func (r *fakeRepo) AddCard(_ context.Context, c *domain.Card) error {
	if r.addErr != nil {
		return r.addErr
	}
	r.added = append(r.added, c)
	return nil
}

func (r *fakeRepo) GetCard(context.Context, domain.CardID) (*domain.Card, error) {
	return nil, domain.ErrCardNotFound
}

var errHash = errors.New("hash failure")

type fakeHasher struct {
	err    error
	gotPAN string
}

func (h *fakeHasher) Hash(pan string) (string, error) {
	h.gotPAN = pan
	if h.err != nil {
		return "", h.err
	}
	return "hash:" + pan, nil
}

func TestCreateCardHandler_Handle(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		cmd     command.CreateCard
		id      domain.CardID
		repoErr error
		hashErr error
		wantErr error
		wantAdd bool
	}{
		{
			name:    "stores the card",
			cmd:     command.CreateCard{PAN: "4111111111111111", CardHolderName: "Ada Lovelace", ExpMonth: 12, ExpYear: 2030},
			id:      "card-1",
			wantAdd: true,
		},
		{
			name:    "domain validation error is returned and nothing is stored",
			cmd:     command.CreateCard{PAN: "123", CardHolderName: "Ada Lovelace", ExpMonth: 12, ExpYear: 2030},
			id:      "card-1",
			wantErr: domain.ErrInvalidPAN,
		},
		{
			name:    "empty id is rejected",
			cmd:     command.CreateCard{PAN: "4111111111111111", CardHolderName: "Ada Lovelace", ExpMonth: 12, ExpYear: 2030},
			id:      "",
			wantErr: domain.ErrIDIsRequired,
		},
		{
			name:    "blank holder name is rejected",
			cmd:     command.CreateCard{PAN: "4111111111111111", CardHolderName: "  ", ExpMonth: 12, ExpYear: 2030},
			id:      "card-1",
			wantErr: domain.ErrCardHolderNameRequired,
		},
		{
			name:    "hasher error is returned and nothing is stored",
			cmd:     command.CreateCard{PAN: "4111111111111111", CardHolderName: "Ada Lovelace", ExpMonth: 12, ExpYear: 2030},
			id:      "card-1",
			hashErr: errHash,
			wantErr: errHash,
		},
		{
			name:    "repository error is returned",
			cmd:     command.CreateCard{PAN: "4111111111111111", CardHolderName: "Ada Lovelace", ExpMonth: 12, ExpYear: 2030},
			id:      "card-1",
			repoErr: domain.ErrCardAlreadyExists,
			wantErr: domain.ErrCardAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{addErr: tt.repoErr}
			h := command.NewCreateCardHandler(repo,
				&fakeHasher{err: tt.hashErr},
				func() domain.CardID { return tt.id },
				func() time.Time { return now },
			)

			got, err := h.Handle(context.Background(), tt.cmd)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				if len(repo.added) != 0 {
					t.Fatalf("stored %d cards, want 0", len(repo.added))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantAdd || len(repo.added) != 1 {
				t.Fatalf("stored %d cards, want 1", len(repo.added))
			}
			stored := repo.added[0]
			if got != stored {
				t.Fatal("returned card is not the stored one")
			}
			if stored.ID() != tt.id {
				t.Errorf("id = %q, want %q", stored.ID(), tt.id)
			}
			if stored.Last4() != "1111" {
				t.Errorf("last4 = %q, want 1111", stored.Last4())
			}
			if !stored.CreatedAt().Equal(now) {
				t.Errorf("createdAt = %v, want %v", stored.CreatedAt(), now)
			}
			if stored.CardHolderName() != "Ada Lovelace" {
				t.Errorf("holder name = %q, want Ada Lovelace", stored.CardHolderName())
			}
			if stored.PANHash() != "hash:4111111111111111" {
				t.Errorf("pan hash = %q, want the hasher output", stored.PANHash())
			}
		})
	}
}

func TestNewCreateCardHandler_PanicsOnNilDependency(t *testing.T) {
	id := func() domain.CardID { return "x" }
	now := func() time.Time { return time.Time{} }
	repo := &fakeRepo{}
	hasher := &fakeHasher{}

	tests := []struct {
		name  string
		build func()
	}{
		{"nil repository", func() { command.NewCreateCardHandler(nil, hasher, id, now) }},
		{"nil hasher", func() { command.NewCreateCardHandler(repo, nil, id, now) }},
		{"nil id generator", func() { command.NewCreateCardHandler(repo, hasher, nil, now) }},
		{"nil clock", func() { command.NewCreateCardHandler(repo, hasher, id, nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			tt.build()
		})
	}
}
