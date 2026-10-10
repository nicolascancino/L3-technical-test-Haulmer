package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

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
	valid := command.CreateCard{PAN: "4111111111111111", CardHolderName: "Ada Lovelace", ExpMonth: 12, ExpYear: 2030}

	tests := []struct {
		name    string
		cmd     command.CreateCard
		repoErr error
		hashErr error
		wantErr error
	}{
		{name: "stores the card", cmd: valid},
		{
			name:    "domain validation error is returned and nothing is stored",
			cmd:     command.CreateCard{PAN: "123", CardHolderName: "Ada Lovelace", ExpMonth: 12, ExpYear: 2030},
			wantErr: domain.ErrInvalidPAN,
		},
		{
			name:    "blank holder name is rejected",
			cmd:     command.CreateCard{PAN: "4111111111111111", CardHolderName: "  ", ExpMonth: 12, ExpYear: 2030},
			wantErr: domain.ErrCardHolderNameRequired,
		},
		{
			name:    "hasher error is returned and nothing is stored",
			cmd:     valid,
			hashErr: errHash,
			wantErr: errHash,
		},
		{
			name:    "repository error is returned",
			cmd:     valid,
			repoErr: domain.ErrCardAlreadyExists,
			wantErr: domain.ErrCardAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{addErr: tt.repoErr}
			h := command.NewCreateCardHandler(repo, &fakeHasher{err: tt.hashErr})

			before := time.Now()
			got, err := h.Handle(context.Background(), tt.cmd)
			after := time.Now()

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
			if len(repo.added) != 1 {
				t.Fatalf("stored %d cards, want 1", len(repo.added))
			}
			stored := repo.added[0]
			if got != stored {
				t.Fatal("returned card is not the stored one")
			}
			id, err := uuid.Parse(string(stored.ID()))
			if err != nil {
				t.Fatalf("id %q is not a UUID: %v", stored.ID(), err)
			}
			if id.Version() != 7 {
				t.Errorf("id version = %d, want 7", id.Version())
			}
			if stored.Last4() != "1111" {
				t.Errorf("last4 = %q, want 1111", stored.Last4())
			}
			for name, ts := range map[string]time.Time{"createdAt": stored.CreatedAt(), "updatedAt": stored.UpdatedAt()} {
				if ts.Before(before.Truncate(time.Nanosecond)) || ts.After(after) {
					t.Errorf("%s = %v, want between %v and %v", name, ts, before, after)
				}
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

func TestCreateCardHandler_Handle_GeneratesDistinctIDs(t *testing.T) {
	h := command.NewCreateCardHandler(&fakeRepo{}, &fakeHasher{})
	cmd := command.CreateCard{PAN: "4111111111111111", CardHolderName: "Ada Lovelace", ExpMonth: 12, ExpYear: 2030}

	first, err := h.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := h.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.ID() == second.ID() {
		t.Errorf("both cards got id %q", first.ID())
	}
}

func TestNewCreateCardHandler_PanicsOnNilDependency(t *testing.T) {
	repo := &fakeRepo{}
	hasher := &fakeHasher{}

	tests := []struct {
		name  string
		build func()
	}{
		{"nil repository", func() { command.NewCreateCardHandler(nil, hasher) }},
		{"nil hasher", func() { command.NewCreateCardHandler(repo, nil) }},
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
