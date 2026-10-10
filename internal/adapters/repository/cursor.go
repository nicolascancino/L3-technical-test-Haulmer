package repository

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

// cursor is the keyset position (created_at, id) of the last row of a page.
type cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// Microsecond precision matches what timestamptz stores.
func encodeCursor(c cursor) string {
	raw := fmt.Sprintf("%d|%s", c.CreatedAt.UnixMicro(), c.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(s string) (cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return cursor{}, domain.ErrInvalidCursor
	}
	micros, id, ok := strings.Cut(string(raw), "|")
	if !ok {
		return cursor{}, domain.ErrInvalidCursor
	}
	usec, err := strconv.ParseInt(micros, 10, 64)
	if err != nil {
		return cursor{}, domain.ErrInvalidCursor
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return cursor{}, domain.ErrInvalidCursor
	}
	return cursor{CreatedAt: time.UnixMicro(usec).UTC(), ID: parsedID}, nil
}
