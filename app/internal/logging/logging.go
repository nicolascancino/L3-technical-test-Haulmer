// Package logging builds the application's slog logger.
package logging

import (
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
)

const (
	service  = "cards-api"
	redacted = "[REDACTED]"
)

var sensitiveKeys = map[string]bool{
	"pan":           true,
	"panhash":       true,
	"pan_hash":      true,
	"password":      true,
	"salt":          true,
	"secret":        true,
	"token":         true,
	"authorization": true,
}

type Options struct {
	Level    slog.Level
	Format   config.LogFormat
	Env      config.Env
	Instance string
}

func New(w io.Writer, opts Options) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{
		Level:       opts.Level,
		AddSource:   opts.Level <= slog.LevelDebug,
		ReplaceAttr: replaceAttr,
	}

	var handler slog.Handler = slog.NewTextHandler(w, handlerOpts)
	if opts.Format == config.LogJSON {
		handler = slog.NewJSONHandler(w, handlerOpts)
	}

	return slog.New(handler).With(
		"service", service,
		"env", string(opts.Env),
		"instance", opts.Instance,
	)
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && a.Key == slog.TimeKey && a.Value.Kind() == slog.KindTime {
		a.Value = slog.StringValue(a.Value.Time().UTC().Format(time.RFC3339))
		return a
	}
	if sensitiveKeys[strings.ToLower(a.Key)] {
		a.Value = slog.StringValue(redacted)
	}
	return a
}
