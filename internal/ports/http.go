package ports

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/command"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/query"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

const (
	maxBodyBytes = 1 << 20
	defaultLimit = 20
	maxLimit     = 100
)

type HttpServer struct {
	app      app.Application
	instance string
}

func NewHttpServer(application app.Application, instance string) HttpServer {
	return HttpServer{app: application, instance: instance}
}

func (h HttpServer) CreateCard(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var body CreateCardJSONRequestBody
	if err := dec.Decode(&body); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "request body exceeds the 1 MiB limit")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}

	card, err := h.app.Commands.CreateCard.Handle(r.Context(), command.CreateCard{
		PAN:            body.Pan,
		CardHolderName: body.CardHolderName,
		ExpMonth:       body.ExpMonth,
		ExpYear:        body.ExpYear,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, Card{
		Id:             string(card.ID()),
		CardHolderName: card.CardHolderName(),
		Brand:          CardBrand(card.Brand()),
		Last4:          card.Last4(),
		ExpMonth:       card.ExpMonth(),
		ExpYear:        card.ExpYear(),
		CreatedAt:      card.CreatedAt(),
		UpdatedAt:      card.UpdatedAt(),
	})
}

func (h HttpServer) GetCard(w http.ResponseWriter, r *http.Request, id string) {
	card, err := h.app.Queries.GetCard.Handle(r.Context(), query.GetCard{ID: id})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toCard(card))
}

func (h HttpServer) ListCards(w http.ResponseWriter, r *http.Request, params ListCardsParams) {
	limit := defaultLimit
	if params.Limit != nil {
		limit = *params.Limit
	}
	if limit < 1 || limit > maxLimit {
		writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 100")
		return
	}

	var cursor string
	if params.Cursor != nil {
		cursor = *params.Cursor
	}

	page, err := h.app.Queries.ListCards.Handle(r.Context(), query.ListCards{Limit: limit, Cursor: cursor})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	out := CardPage{Items: toCards(page.Items)}
	if page.NextCursor != "" {
		out.NextCursor = &page.NextCursor
	}
	writeJSON(w, http.StatusOK, out)
}

func (h HttpServer) SearchCards(w http.ResponseWriter, r *http.Request, params SearchCardsParams) {
	if !isLast4(params.Last4) {
		writeError(w, http.StatusBadRequest, "invalid_last4", domain.ErrInvalidLast4.Error())
		return
	}

	cards, err := h.app.Queries.SearchCards.Handle(r.Context(), query.SearchCards{Last4: params.Last4})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, CardList{Items: toCards(cards)})
}

func (h HttpServer) GetHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, Health{Status: "ok", Instance: h.instance})
}

func isLast4(s string) bool {
	if len(s) != 4 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func toCard(c query.Card) Card {
	return Card{
		Id: c.ID, Last4: c.Last4, ExpMonth: c.ExpMonth, ExpYear: c.ExpYear,
		CardHolderName: c.CardHolderName, Brand: CardBrand(c.Brand),
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func toCards(cards []query.Card) []Card {
	out := make([]Card, 0, len(cards))
	for _, c := range cards {
		out = append(out, toCard(c))
	}
	return out
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidPAN):
		writeError(w, http.StatusUnprocessableEntity, "invalid_pan", domain.ErrInvalidPAN.Error())
	case errors.Is(err, domain.ErrCardHolderNameRequired):
		writeError(w, http.StatusUnprocessableEntity, "card_holder_name_required", domain.ErrCardHolderNameRequired.Error())
	case errors.Is(err, domain.ErrInvalidExpiry):
		writeError(w, http.StatusUnprocessableEntity, "invalid_expiry", domain.ErrInvalidExpiry.Error())
	case errors.Is(err, domain.ErrCardExpired):
		writeError(w, http.StatusUnprocessableEntity, "card_expired", domain.ErrCardExpired.Error())
	case errors.Is(err, domain.ErrCardNotFound):
		writeError(w, http.StatusNotFound, "card_not_found", domain.ErrCardNotFound.Error())
	case errors.Is(err, domain.ErrInvalidCursor):
		writeError(w, http.StatusBadRequest, "invalid_cursor", domain.ErrInvalidCursor.Error())
	default:
		slog.ErrorContext(r.Context(), "unexpected error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}

func writeError(w http.ResponseWriter, status int, slug, message string) {
	writeJSON(w, status, Error{Slug: slug, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode response", "error", err)
	}
}
