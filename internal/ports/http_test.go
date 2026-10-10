package ports_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/command"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/query"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/ports"
)

const validPAN = "4111111111111111"

var createdAt = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

type fakeRepo struct{ err error }

func (r *fakeRepo) AddCard(context.Context, *domain.Card) error { return r.err }

func (r *fakeRepo) GetCard(context.Context, domain.CardID) (*domain.Card, error) {
	return nil, domain.ErrCardNotFound
}

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

var sampleCard = query.Card{ID: "card-1", Last4: "1111", ExpMonth: 12, ExpYear: 2099, CreatedAt: createdAt}

func newHandler(repo *fakeRepo, reader *fakeReader) http.Handler {
	application := app.Application{
		Commands: app.Commands{
			CreateCard: command.NewCreateCardHandler(
				repo,
				func() domain.CardID { return "card-1" },
				func() time.Time { return createdAt },
			),
		},
		Queries: app.Queries{
			GetCard:     query.NewGetCardHandler(reader),
			ListCards:   query.NewListCardsHandler(reader),
			SearchCards: query.NewSearchCardsHandler(reader),
		},
	}
	return ports.HandlerFromMux(ports.NewHttpServer(application, "pod-1"), chi.NewRouter())
}

func do(h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func assertResponse(t *testing.T, rec *httptest.ResponseRecorder, status int, contains ...string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, status, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content type = %q, want application/json", ct)
	}
	for _, s := range contains {
		if !strings.Contains(rec.Body.String(), s) {
			t.Errorf("body %s does not contain %s", rec.Body, s)
		}
	}
}

func TestCreateCard(t *testing.T) {
	boom := errors.New("pq: connection secret-host refused")
	tests := []struct {
		name   string
		body   string
		repo   error
		status int
		want   []string
	}{
		{
			name:   "creates a card",
			body:   `{"pan":"` + validPAN + `","expMonth":12,"expYear":2099}`,
			status: http.StatusCreated,
			want:   []string{`"id":"card-1"`, `"last4":"1111"`, `"expMonth":12`, `"expYear":2099`, `"createdAt":"2026-01-15T10:00:00Z"`},
		},
		{name: "malformed json", body: `{`, status: http.StatusBadRequest, want: []string{`"slug":"invalid_json"`}},
		{
			name:   "unknown field",
			body:   `{"pan":"` + validPAN + `","expMonth":12,"expYear":2099,"cvv":"123"}`,
			status: http.StatusBadRequest,
			want:   []string{`"slug":"invalid_json"`},
		},
		{
			name:   "invalid pan",
			body:   `{"pan":"123","expMonth":12,"expYear":2099}`,
			status: http.StatusUnprocessableEntity,
			want:   []string{`"slug":"invalid_pan"`},
		},
		{
			name:   "invalid expiry",
			body:   `{"pan":"` + validPAN + `","expMonth":12,"expYear":2099}`,
			repo:   domain.ErrInvalidExpiry,
			status: http.StatusUnprocessableEntity,
			want:   []string{`"slug":"invalid_expiry"`},
		},
		{
			name:   "expired card",
			body:   `{"pan":"` + validPAN + `","expMonth":12,"expYear":2099}`,
			repo:   domain.ErrCardExpired,
			status: http.StatusUnprocessableEntity,
			want:   []string{`"slug":"card_expired"`},
		},
		{
			name:   "unexpected repository failure",
			body:   `{"pan":"` + validPAN + `","expMonth":12,"expYear":2099}`,
			repo:   boom,
			status: http.StatusInternalServerError,
			want:   []string{`"slug":"internal"`, `"message":"internal error"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(newHandler(&fakeRepo{err: tt.repo}, &fakeReader{}), http.MethodPost, "/cards", tt.body)

			assertResponse(t, rec, tt.status, tt.want...)
			if strings.Contains(rec.Body.String(), validPAN) {
				t.Error("response leaks the PAN")
			}
			if strings.Contains(rec.Body.String(), "secret-host") {
				t.Error("response leaks the raw error")
			}
		})
	}
}

func TestGetCard(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   []string
	}{
		{name: "hit", status: http.StatusOK, want: []string{`"id":"card-1"`, `"last4":"1111"`}},
		{name: "not found", err: domain.ErrCardNotFound, status: http.StatusNotFound, want: []string{`"slug":"card_not_found"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeReader{card: sampleCard, err: tt.err}
			rec := do(newHandler(&fakeRepo{}, reader), http.MethodGet, "/cards/card-1", "")

			assertResponse(t, rec, tt.status, tt.want...)
			if reader.gotID != "card-1" {
				t.Errorf("id = %q, want card-1", reader.gotID)
			}
		})
	}
}

func TestListCards(t *testing.T) {
	next := query.CardPage{Items: []query.Card{sampleCard}, NextCursor: "abc"}
	tests := []struct {
		name       string
		target     string
		page       query.CardPage
		err        error
		status     int
		want       []string
		notWant    string
		wantLimit  int
		wantCursor string
		reached    bool
	}{
		{name: "default limit", target: "/cards", status: 200, want: []string{`"items":[]`}, notWant: "nextCursor", wantLimit: 20, reached: true},
		{name: "limit 100", target: "/cards?limit=100", status: 200, wantLimit: 100, reached: true},
		{name: "limit 0", target: "/cards?limit=0", status: 400, want: []string{`"slug":"invalid_limit"`}},
		{name: "limit 101", target: "/cards?limit=101", status: 400, want: []string{`"slug":"invalid_limit"`}},
		{
			name: "cursor and next cursor", target: "/cards?cursor=xyz", page: next, status: 200,
			want: []string{`"nextCursor":"abc"`, `"id":"card-1"`}, wantLimit: 20, wantCursor: "xyz", reached: true,
		},
		{name: "empty cursor is first page", target: "/cards?cursor=", status: 200, wantLimit: 20, reached: true},
		{
			name: "invalid cursor", target: "/cards?cursor=bad", err: domain.ErrInvalidCursor, status: 400,
			want: []string{`"slug":"invalid_cursor"`}, wantLimit: 20, wantCursor: "bad", reached: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeReader{page: tt.page, err: tt.err}
			rec := do(newHandler(&fakeRepo{}, reader), http.MethodGet, tt.target, "")

			assertResponse(t, rec, tt.status, tt.want...)
			if tt.notWant != "" && strings.Contains(rec.Body.String(), tt.notWant) {
				t.Errorf("body %s contains %s", rec.Body, tt.notWant)
			}
			if tt.reached && (reader.gotLimit != tt.wantLimit || reader.gotCursor != tt.wantCursor) {
				t.Errorf("reader args = (%d, %q), want (%d, %q)", reader.gotLimit, reader.gotCursor, tt.wantLimit, tt.wantCursor)
			}
			if !tt.reached && reader.gotLimit != 0 {
				t.Error("reader must not be reached")
			}
		})
	}
}

func TestSearchCards(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		cards   []query.Card
		status  int
		want    []string
		reached bool
	}{
		{name: "valid", target: "/cards:search?last4=1234", cards: []query.Card{sampleCard}, status: 200, want: []string{`"id":"card-1"`}, reached: true},
		{name: "empty result", target: "/cards:search?last4=1234", status: 200, want: []string{`"items":[]`}, reached: true},
		{name: "non digit", target: "/cards:search?last4=12a4", status: 400, want: []string{`"slug":"invalid_last4"`}},
		{name: "too short", target: "/cards:search?last4=123", status: 400, want: []string{`"slug":"invalid_last4"`}},
		{name: "too long", target: "/cards:search?last4=12345", status: 400, want: []string{`"slug":"invalid_last4"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeReader{cards: tt.cards, card: query.Card{ID: "wrong"}}
			rec := do(newHandler(&fakeRepo{}, reader), http.MethodGet, tt.target, "")

			assertResponse(t, rec, tt.status, tt.want...)
			if tt.reached && reader.gotLast4 != "1234" {
				t.Errorf("last4 = %q, want 1234", reader.gotLast4)
			}
			if reader.gotID != "" {
				t.Error("search was routed to get card")
			}
			if !tt.reached && reader.gotLast4 != "" {
				t.Error("reader must not be reached")
			}
		})
	}
}

func TestGetHealth(t *testing.T) {
	rec := do(newHandler(&fakeRepo{}, &fakeReader{}), http.MethodGet, "/health", "")

	assertResponse(t, rec, http.StatusOK, `"status":"ok"`, `"instance":"pod-1"`)
}
