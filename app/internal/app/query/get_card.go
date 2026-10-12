package query

import (
	"context"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

type GetCard struct {
	ID string
}

type GetCardHandler struct {
	reader CardReader
}

func NewGetCardHandler(reader CardReader) GetCardHandler {
	if reader == nil {
		panic("nil card reader")
	}
	return GetCardHandler{reader: reader}
}

func (h GetCardHandler) Handle(ctx context.Context, q GetCard) (Card, error) {
	return h.reader.GetCard(ctx, domain.CardID(q.ID))
}
