package query

import "context"

type SearchCards struct {
	CardFilter
	Limit  int
	Cursor string
}

type SearchCardsHandler struct {
	reader CardReader
}

func NewSearchCardsHandler(reader CardReader) SearchCardsHandler {
	if reader == nil {
		panic("nil card reader")
	}
	return SearchCardsHandler{reader: reader}
}

func (h SearchCardsHandler) Handle(ctx context.Context, q SearchCards) (CardPage, error) {
	return h.reader.SearchCards(ctx, q.CardFilter, q.Limit, q.Cursor)
}
