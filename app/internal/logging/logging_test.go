package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
)

func options(level slog.Level, format config.LogFormat) Options {
	return Options{Level: level, Format: format, Env: config.Cloud, Instance: "cards-api-0"}
}

func TestNew_FiltersByLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, options(slog.LevelWarn, config.LogText))

	logger.Info("hidden")
	logger.Warn("shown")

	if strings.Contains(buf.String(), "hidden") || !strings.Contains(buf.String(), "shown") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestNew_AddsFixedAttributes(t *testing.T) {
	for _, format := range []config.LogFormat{config.LogText, config.LogJSON} {
		var buf bytes.Buffer
		New(&buf, options(slog.LevelInfo, format)).Info("started")

		for _, want := range []string{"cards-api", "cloud", "cards-api-0"} {
			if !strings.Contains(buf.String(), want) {
				t.Errorf("%s output %q does not contain %q", format, buf.String(), want)
			}
		}
	}
}

func TestNew_JSONUsesFixedKeysAndUTCTime(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, options(slog.LevelInfo, config.LogJSON)).Info("started")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("output is not JSON: %v", err)
	}
	if record["service"] != "cards-api" || record["env"] != "cloud" || record["instance"] != "cards-api-0" {
		t.Errorf("record = %v", record)
	}
	if got, _ := record["time"].(string); !strings.HasSuffix(got, "Z") {
		t.Errorf("time = %q, want UTC", got)
	}
}

func TestNew_MasksSensitiveValues(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, options(slog.LevelInfo, config.LogJSON))

	logger.Info("request",
		"PAN", "4111111111111111",
		"pan_hash", "hash-value",
		"Authorization", "Bearer abc",
		slog.Group("card", slog.String("panhash", "nested-hash"), slog.String("last4", "1111")),
	)

	out := buf.String()
	for _, leaked := range []string{"4111111111111111", "hash-value", "Bearer abc", "nested-hash"} {
		if strings.Contains(out, leaked) {
			t.Errorf("output %q leaks %q", out, leaked)
		}
	}
	if !strings.Contains(out, "[REDACTED]") || !strings.Contains(out, "1111") {
		t.Errorf("output = %q", out)
	}
}

func TestNew_AddsSourceOnlyAtDebug(t *testing.T) {
	var debug, info bytes.Buffer
	New(&debug, options(slog.LevelDebug, config.LogJSON)).Debug("probe")
	New(&info, options(slog.LevelInfo, config.LogJSON)).Info("probe")

	if !strings.Contains(debug.String(), `"source"`) {
		t.Errorf("debug output has no source: %q", debug.String())
	}
	if strings.Contains(info.String(), `"source"`) {
		t.Errorf("info output has source: %q", info.String())
	}
}
