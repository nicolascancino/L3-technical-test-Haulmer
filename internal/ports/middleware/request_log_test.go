package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/logging"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/ports/middleware"
)

func serve(t *testing.T, method, target, body string) (statusLog map[string]any, output string) {
	t.Helper()
	var buf bytes.Buffer
	logger := logging.New(&buf, logging.Options{Level: slog.LevelInfo, Format: config.LogJSON, Env: config.Cloud, Instance: "test"})

	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.RequestLog(logger))
	router.Get("/cards/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	router.Post("/cards", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) })
	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	router.Get("/boom", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) })

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	router.ServeHTTP(httptest.NewRecorder(), req)

	output = buf.String()
	if output == "" {
		return nil, output
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &statusLog); err != nil {
		t.Fatalf("output is not a single JSON line: %q: %v", output, err)
	}
	return statusLog, output
}

func TestRequestLog_LogsRouteMetricsAndEMF(t *testing.T) {
	line, _ := serve(t, http.MethodGet, "/cards/123456?secret=1", "")

	if line["msg"] != "request" || line["level"] != "INFO" {
		t.Fatalf("msg/level = %v/%v", line["msg"], line["level"])
	}
	if id, _ := line["request_id"].(string); id == "" {
		t.Error("request_id is empty")
	}
	if line["method"] != "GET" || line["route"] != "/cards/{id}" || line["status"] != float64(200) ||
		line["status_class"] != "2xx" || line["requests"] != float64(1) {
		t.Errorf("unexpected attributes: %v", line)
	}
	if latency, ok := line["latency_ms"].(float64); !ok || latency < 0 {
		t.Errorf("latency_ms = %v", line["latency_ms"])
	}

	aws, ok := line["_aws"].(map[string]any)
	if !ok {
		t.Fatalf("_aws = %v", line["_aws"])
	}
	directive := aws["CloudWatchMetrics"].([]any)[0].(map[string]any)
	dimensions := directive["Dimensions"].([]any)[0].([]any)
	if len(dimensions) != 3 || dimensions[0] != "method" || dimensions[1] != "route" || dimensions[2] != "status_class" {
		t.Errorf("dimensions = %v", dimensions)
	}
	metrics := directive["Metrics"].([]any)
	if len(metrics) != 2 || metrics[0].(map[string]any)["Name"] != "requests" || metrics[1].(map[string]any)["Name"] != "latency_ms" {
		t.Errorf("metrics = %v", metrics)
	}
}

func TestRequestLog_UnknownRouteIsUnmatched(t *testing.T) {
	line, _ := serve(t, http.MethodGet, "/nope/42", "")

	if line["route"] != "unmatched" || line["status"] != float64(404) || line["status_class"] != "4xx" {
		t.Errorf("unexpected attributes: %v", line)
	}
}

func TestRequestLog_HealthIsSilentAtInfo(t *testing.T) {
	if _, output := serve(t, http.MethodGet, "/health", ""); output != "" {
		t.Errorf("output = %q, want empty", output)
	}
}

func TestRequestLog_ServerErrorIsLoggedAtError(t *testing.T) {
	line, _ := serve(t, http.MethodGet, "/boom", "")

	if line["level"] != "ERROR" || line["status"] != float64(500) || line["status_class"] != "5xx" {
		t.Errorf("unexpected attributes: %v", line)
	}
}

func TestRequestLog_NeverLogsRequestBody(t *testing.T) {
	_, output := serve(t, http.MethodPost, "/cards", `{"pan":"4111111111111111"}`)

	if output == "" || strings.Contains(output, "4111111111111111") {
		t.Errorf("output = %q", output)
	}
}

func TestRequestLog_LogsRawPathButNotQuery(t *testing.T) {
	line, output := serve(t, http.MethodGet, "/cards/123456?secret=1", "")

	if line["path"] != "/cards/123456" || line["route"] != "/cards/{id}" {
		t.Errorf("path/route = %v/%v", line["path"], line["route"])
	}
	if strings.Contains(output, "secret") {
		t.Errorf("query string leaked: %q", output)
	}
	dimensions := line["_aws"].(map[string]any)["CloudWatchMetrics"].([]any)[0].(map[string]any)["Dimensions"].([]any)[0].([]any)
	for _, d := range dimensions {
		if d == "path" {
			t.Error("path must not be an EMF dimension")
		}
	}
}
