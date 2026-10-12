package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/ports/middleware"
)

func requestWithID(id *string) (header, inContext string) {
	var got string
	handler := middleware.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = middleware.RequestIDFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if id != nil {
		req.Header.Set("X-Request-Id", *id)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Header().Get("X-Request-Id"), got
}

func TestRequestID_KeepsValidIncomingID(t *testing.T) {
	for _, id := range []string{"abc-123", "A.b_C-9", strings.Repeat("a", 64), "a"} {
		header, inContext := requestWithID(&id)
		if header != id || inContext != id {
			t.Errorf("id %q: header = %q, context = %q", id, header, inContext)
		}
	}
}

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	header, inContext := requestWithID(nil)

	if header == "" || header != inContext {
		t.Errorf("header = %q, context = %q", header, inContext)
	}
}

func TestRequestID_ReplacesInvalidIncomingID(t *testing.T) {
	invalid := []string{
		strings.Repeat("a", 65),
		"has space",
		"line\nbreak",
		"carriage\rreturn",
		"",
		"   ",
		"a/b",
		"<script>",
	}
	for _, id := range invalid {
		header, inContext := requestWithID(&id)
		if header == "" || header == id || header != inContext {
			t.Errorf("id %q: header = %q, context = %q", id, header, inContext)
		}
	}
}

func TestRequestID_GeneratedIDsDiffer(t *testing.T) {
	first, _ := requestWithID(nil)
	second, _ := requestWithID(nil)

	if first == second {
		t.Errorf("both requests got %q", first)
	}
}

func TestRequestIDFromContext_EmptyWhenAbsent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if id := middleware.RequestIDFromContext(req.Context()); id != "" {
		t.Errorf("id = %q", id)
	}
}
