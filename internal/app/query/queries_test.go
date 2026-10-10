package query_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/query"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

type fakeReader struct {
	card  query.Card
	page  query.CardPage
	cards []query.Card
	err   error

	gotID     domain.CardID
	gotLimit  int
	gotCursor string
	gotLast4  string
}

func (r *fakeReader) GetCard(_ context.Context, id domain.CardID) (query.Card, error) {
	r.gotID = id
	return r.card, r.err
}

func (r *fakeReader) ListCards(_ context.Context, limit int, cursor string) (query.CardPage, error) {
	r.gotLimit, r.gotCursor = limit, cursor
	return r.page, r.err
}

func (r *fakeReader) SearchByLast4(_ context.Context, last4 string) ([]query.Card, error) {
	r.gotLast4 = last4
	return r.cards, r.err
}

var sample = query.Card{
	ID:        "card-1",
	Last4:     "1111",
	ExpMonth:  12,
	ExpYear:   2030,
	CreatedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
}

func TestGetCardHandler_Handle(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want    query.Card
		wantErr error
	}{
		{name: "delegates to the reader", want: sample},
		{name: "propagates not found", err: domain.ErrCardNotFound, wantErr: domain.ErrCardNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeReader{card: tt.want, err: tt.err}
			got, err := query.NewGetCardHandler(reader).Handle(context.Background(), query.GetCard{ID: "card-1"})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if reader.gotID != "card-1" {
				t.Errorf("id = %q, want card-1", reader.gotID)
			}
			if got != tt.want {
				t.Errorf("card = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListCardsHandler_Handle(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name    string
		err     error
		page    query.CardPage
		wantErr error
	}{
		{name: "delegates to the reader", page: query.CardPage{Items: []query.Card{sample}, NextCursor: "next"}},
		{name: "propagates reader error", err: boom, wantErr: boom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeReader{page: tt.page, err: tt.err}
			got, err := query.NewListCardsHandler(reader).Handle(context.Background(), query.ListCards{Limit: 10, Cursor: "abc"})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if reader.gotLimit != 10 || reader.gotCursor != "abc" {
				t.Errorf("args = (%d, %q), want (10, abc)", reader.gotLimit, reader.gotCursor)
			}
			if !reflect.DeepEqual(got, tt.page) {
				t.Errorf("page = %+v, want %+v", got, tt.page)
			}
		})
	}
}

func TestSearchCardsHandler_Handle(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name    string
		err     error
		cards   []query.Card
		wantErr error
	}{
		{name: "delegates to the reader", cards: []query.Card{sample}},
		{name: "propagates reader error", err: boom, wantErr: boom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeReader{cards: tt.cards, err: tt.err}
			got, err := query.NewSearchCardsHandler(reader).Handle(context.Background(), query.SearchCards{Last4: "1111"})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if reader.gotLast4 != "1111" {
				t.Errorf("last4 = %q, want 1111", reader.gotLast4)
			}
			if !reflect.DeepEqual(got, tt.cards) {
				t.Errorf("cards = %+v, want %+v", got, tt.cards)
			}
		})
	}
}

func TestConstructors_PanicOnNilReader(t *testing.T) {
	tests := []struct {
		name  string
		build func()
	}{
		{"get card", func() { query.NewGetCardHandler(nil) }},
		{"list cards", func() { query.NewListCardsHandler(nil) }},
		{"search cards", func() { query.NewSearchCardsHandler(nil) }},
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
