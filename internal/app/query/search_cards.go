package query

import "context"

type SearchCards struct {
	Last4 string
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

func (h SearchCardsHandler) Handle(ctx context.Context, q SearchCards) ([]Card, error) {
	return h.reader.SearchByLast4(ctx, q.Last4)
}
