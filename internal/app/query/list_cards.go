package query

import "context"

type ListCards struct {
	Limit  int
	Cursor string
}

type ListCardsHandler struct {
	reader CardReader
}

func NewListCardsHandler(reader CardReader) ListCardsHandler {
	if reader == nil {
		panic("nil card reader")
	}
	return ListCardsHandler{reader: reader}
}

func (h ListCardsHandler) Handle(ctx context.Context, q ListCards) (CardPage, error) {
	return h.reader.ListCards(ctx, q.Limit, q.Cursor)
}
