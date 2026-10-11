package repository

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/logging"
)

func newTestLogger(level slog.Level) (gormLogger, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := logging.New(&buf, logging.Options{Level: level, Format: config.LogJSON, Env: config.Local, Instance: "test-0"})
	return NewGormLogger(logger).(gormLogger), &buf
}

func traceOf(sql string) func() (string, int64) {
	return func() (string, int64) { return sql, 1 }
}

func TestGormLogger_FailedInsertDoesNotLogBoundValues(t *testing.T) {
	l, buf := newTestLogger(slog.LevelInfo)
	const secret = "secret-pan-hash-value"

	sql, vars := l.ParamsFilter(context.Background(), "INSERT INTO cards (pan_hash) VALUES ($1)", secret)
	if len(vars) != 0 {
		t.Errorf("vars = %v, want none", vars)
	}
	l.Trace(context.Background(), time.Now(), traceOf(sql), errors.New("boom"))

	out := buf.String()
	if !strings.Contains(out, "query failed") || !strings.Contains(out, "$1") {
		t.Errorf("output = %q", out)
	}
	if strings.Contains(out, secret) {
		t.Errorf("output leaks the bound value: %q", out)
	}
}

func TestGormLogger_IgnoresNotFoundAndDuplicatedKey(t *testing.T) {
	for _, err := range []error{gorm.ErrRecordNotFound, gorm.ErrDuplicatedKey} {
		for _, level := range []slog.Level{slog.LevelInfo, slog.LevelDebug} {
			l, buf := newTestLogger(level)
			l.Trace(context.Background(), time.Now(), traceOf("SELECT 1"), err)
			if buf.Len() != 0 {
				t.Errorf("%v at %v: output = %q, want none", err, level, buf.String())
			}
		}
	}
}

func TestGormLogger_SlowQueryWarns(t *testing.T) {
	l, buf := newTestLogger(slog.LevelInfo)

	l.Trace(context.Background(), time.Now().Add(-time.Second), traceOf("SELECT 1"), nil)

	out := buf.String()
	if !strings.Contains(out, `"level":"WARN"`) || !strings.Contains(out, "slow query") {
		t.Errorf("output = %q", out)
	}
}

func TestGormLogger_FastQueryLogsOnlyAtDebug(t *testing.T) {
	t.Run("info", func(t *testing.T) {
		l, buf := newTestLogger(slog.LevelInfo)
		calls := 0
		l.Trace(context.Background(), time.Now(), func() (string, int64) {
			calls++
			return "SELECT 1", 1
		}, nil)
		if buf.Len() != 0 || calls != 0 {
			t.Errorf("output = %q, fc calls = %d, want none", buf.String(), calls)
		}
	})
	t.Run("debug", func(t *testing.T) {
		l, buf := newTestLogger(slog.LevelDebug)
		l.Trace(context.Background(), time.Now(), traceOf("SELECT 1"), nil)
		out := buf.String()
		if !strings.Contains(out, `"level":"DEBUG"`) || !strings.Contains(out, `"msg":"query"`) {
			t.Errorf("output = %q", out)
		}
	})
}

func TestGormLogger_UsesSharedLoggerAttributes(t *testing.T) {
	l, buf := newTestLogger(slog.LevelInfo)

	l.Error(context.Background(), "connection %s", "lost")

	out := buf.String()
	for _, want := range []string{"cards-api", "local", "test-0", "connection lost"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q does not contain %q", out, want)
		}
	}
}
