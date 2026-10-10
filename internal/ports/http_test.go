package ports_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type fakeReader struct {
	card query.Card
	page query.CardPage
	err  error

	gotID     domain.CardID
	gotLimit  int
	gotCursor string
	gotFilter query.CardFilter
}

func (r *fakeReader) GetCard(_ context.Context, id domain.CardID) (query.Card, error) {
	r.gotID = id
	return r.card, r.err
}

func (r *fakeReader) ListCards(_ context.Context, limit int, cursor string) (query.CardPage, error) {
	r.gotLimit, r.gotCursor = limit, cursor
	return r.page, r.err
}

func (r *fakeReader) SearchCards(_ context.Context, filter query.CardFilter, limit int, cursor string) (query.CardPage, error) {
	r.gotFilter, r.gotLimit, r.gotCursor = filter, limit, cursor
	return r.page, r.err
}

var sampleCard = query.Card{
	ID: "card-1", Last4: "1111", ExpMonth: 12, ExpYear: 2099, CardHolderName: "Ada Lovelace", Brand: "visa",
	CreatedAt: createdAt, UpdatedAt: createdAt.Add(time.Hour),
}

type fakeHasher struct{}

func (fakeHasher) Hash(pan string) (string, error) { return "secret-hash-" + pan, nil }

func newHandler(repo *fakeRepo, reader *fakeReader) http.Handler {
	application := app.Application{
		Commands: app.Commands{
			CreateCard: command.NewCreateCardHandler(repo, fakeHasher{}),
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
			body:   `{"pan":"` + validPAN + `","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2099}`,
			status: http.StatusCreated,
			want:   []string{`"id":"`, `"last4":"1111"`, `"expMonth":12`, `"expYear":2099`, `"cardHolderName":"Ada Lovelace"`, `"brand":"unknown"`, `"createdAt":"`, `"updatedAt":"`},
		},
		{
			name:   "blank holder name",
			body:   `{"pan":"` + validPAN + `","cardHolderName":" ","expMonth":12,"expYear":2099}`,
			status: http.StatusUnprocessableEntity,
			want:   []string{`"slug":"card_holder_name_required"`},
		},
		{name: "malformed json", body: `{`, status: http.StatusBadRequest, want: []string{`"slug":"invalid_json"`}},
		{
			name:   "body over the size limit",
			body:   `{"pan":"` + strings.Repeat("1", 1<<20+1) + `","expMonth":12,"expYear":2099}`,
			status: http.StatusRequestEntityTooLarge,
			want:   []string{`"slug":"payload_too_large"`},
		},
		{
			name:   "unknown field",
			body:   `{"pan":"` + validPAN + `","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2099,"cvv":"123"}`,
			status: http.StatusBadRequest,
			want:   []string{`"slug":"invalid_json"`},
		},
		{
			name:   "invalid pan",
			body:   `{"pan":"123","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2099}`,
			status: http.StatusUnprocessableEntity,
			want:   []string{`"slug":"invalid_pan"`},
		},
		{
			name:   "invalid expiry",
			body:   `{"pan":"` + validPAN + `","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2099}`,
			repo:   domain.ErrInvalidExpiry,
			status: http.StatusUnprocessableEntity,
			want:   []string{`"slug":"invalid_expiry"`},
		},
		{
			name:   "expired card",
			body:   `{"pan":"` + validPAN + `","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2099}`,
			repo:   domain.ErrCardExpired,
			status: http.StatusUnprocessableEntity,
			want:   []string{`"slug":"card_expired"`},
		},
		{
			name:   "duplicate card",
			body:   `{"pan":"` + validPAN + `","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2099}`,
			repo:   fmt.Errorf("%w: 0190-secret-card-id", domain.ErrCardAlreadyExists),
			status: http.StatusConflict,
			want:   []string{`"slug":"card_already_exists"`, `"message":"card already exists"`},
		},
		{
			name:   "unexpected repository failure",
			body:   `{"pan":"` + validPAN + `","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2099}`,
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
			if strings.Contains(rec.Body.String(), "secret-hash") {
				t.Error("response leaks the PAN hash")
			}
			if strings.Contains(rec.Body.String(), "secret-card-id") {
				t.Error("response leaks the card id")
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
		{name: "hit", status: http.StatusOK, want: []string{`"id":"card-1"`, `"last4":"1111"`, `"cardHolderName":"Ada Lovelace"`, `"brand":"visa"`, `"updatedAt":"2026-01-15T11:00:00Z"`}},
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
	const base = "/cards:search?last4=1234"
	next := query.CardPage{Items: []query.Card{sampleCard}, NextCursor: "abc"}
	invalid := func(slug string) []string { return []string{`"slug":"` + slug + `"`} }
	tests := []struct {
		name       string
		target     string
		page       query.CardPage
		err        error
		status     int
		want       []string
		wantFilter query.CardFilter
		wantLimit  int
		wantCursor string
		reached    bool
	}{
		{
			name: "last4 only uses default limit", target: base, status: 200, want: []string{`"items":[]`},
			wantFilter: query.CardFilter{Last4: "1234"}, wantLimit: 20, reached: true,
		},
		{
			name: "page with next cursor", target: base, page: next, status: 200,
			want:       []string{`"id":"card-1"`, `"nextCursor":"abc"`},
			wantFilter: query.CardFilter{Last4: "1234"}, wantLimit: 20, reached: true,
		},
		{
			name:   "all filters",
			target: base + "&brand=visa&cardHolderName=Ada&expMonth=12&expYear=2030&limit=50&cursor=xyz",
			status: 200,
			wantFilter: query.CardFilter{
				Last4: "1234", Brand: "visa", CardHolderName: "Ada", ExpMonth: 12, ExpYear: 2030,
			},
			wantLimit: 50, wantCursor: "xyz", reached: true,
		},
		{
			name: "boundary values", target: base + "&expMonth=1&expYear=9999&limit=100", status: 200,
			wantFilter: query.CardFilter{Last4: "1234", ExpMonth: 1, ExpYear: 9999}, wantLimit: 100, reached: true,
		},
		{
			name: "holder name of 100 characters", target: base + "&cardHolderName=" + strings.Repeat("a", 100), status: 200,
			wantFilter: query.CardFilter{Last4: "1234", CardHolderName: strings.Repeat("a", 100)}, wantLimit: 20, reached: true,
		},
		{
			name: "invalid cursor", target: base + "&cursor=bad", err: domain.ErrInvalidCursor, status: 400,
			want: invalid("invalid_cursor"), wantFilter: query.CardFilter{Last4: "1234"}, wantLimit: 20, wantCursor: "bad", reached: true,
		},
		{name: "non digit", target: "/cards:search?last4=12a4", status: 400, want: invalid("invalid_last4")},
		{name: "too short", target: "/cards:search?last4=123", status: 400, want: invalid("invalid_last4")},
		{name: "too long", target: "/cards:search?last4=12345", status: 400, want: invalid("invalid_last4")},
		{name: "unknown brand", target: base + "&brand=maestro", status: 400, want: invalid("invalid_brand")},
		{name: "blank holder name", target: base + "&cardHolderName=%20%20", status: 400, want: invalid("invalid_card_holder_name")},
		{name: "empty holder name", target: base + "&cardHolderName=", status: 400, want: invalid("invalid_card_holder_name")},
		{name: "holder name too long", target: base + "&cardHolderName=" + strings.Repeat("a", 101), status: 400, want: invalid("invalid_card_holder_name")},
		{name: "expMonth 0", target: base + "&expMonth=0", status: 400, want: invalid("invalid_exp_month")},
		{name: "expMonth 13", target: base + "&expMonth=13", status: 400, want: invalid("invalid_exp_month")},
		{name: "expYear 0", target: base + "&expYear=0", status: 400, want: invalid("invalid_exp_year")},
		{name: "expYear 10000", target: base + "&expYear=10000", status: 400, want: invalid("invalid_exp_year")},
		{name: "limit 0", target: base + "&limit=0", status: 400, want: invalid("invalid_limit")},
		{name: "limit 101", target: base + "&limit=101", status: 400, want: invalid("invalid_limit")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeReader{page: tt.page, err: tt.err, card: query.Card{ID: "wrong"}}
			rec := do(newHandler(&fakeRepo{}, reader), http.MethodGet, tt.target, "")

			assertResponse(t, rec, tt.status, tt.want...)
			if reader.gotID != "" {
				t.Error("search was routed to get card")
			}
			if !tt.reached {
				if reader.gotFilter != (query.CardFilter{}) || reader.gotLimit != 0 {
					t.Error("reader must not be reached")
				}
				return
			}
			if reader.gotFilter != tt.wantFilter {
				t.Errorf("filter = %+v, want %+v", reader.gotFilter, tt.wantFilter)
			}
			if reader.gotLimit != tt.wantLimit || reader.gotCursor != tt.wantCursor {
				t.Errorf("reader args = (%d, %q), want (%d, %q)", reader.gotLimit, reader.gotCursor, tt.wantLimit, tt.wantCursor)
			}
		})
	}
}

func TestGetHealth(t *testing.T) {
	rec := do(newHandler(&fakeRepo{}, &fakeReader{}), http.MethodGet, "/health", "")

	assertResponse(t, rec, http.StatusOK, `"status":"ok"`, `"instance":"pod-1"`)
}

func TestSearchCards_Last4IsRequired(t *testing.T) {
	reader := &fakeReader{}
	rec := do(newHandler(&fakeRepo{}, reader), http.MethodGet, "/cards:search?brand=visa", "")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if reader.gotLimit != 0 {
		t.Error("reader must not be reached")
	}
}

func TestNewHandler_WrapperErrorsAreJSON(t *testing.T) {
	handler := ports.NewHandler(app.Application{}, "test")

	for _, target := range []string{"/cards:search", "/cards?limit=abc"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", target, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("%s: content type = %q, want application/json", target, ct)
		}
		var body ports.Error
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil || body.Slug != "invalid_request" {
			t.Errorf("%s: body = %+v, err = %v", target, body, err)
		}
	}
}
