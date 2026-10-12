package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const slowQueryThreshold = 200 * time.Millisecond

type gormLogger struct {
	logger *slog.Logger
}

func NewGormLogger(logger *slog.Logger) gormlogger.Interface {
	return gormLogger{logger: logger}
}

func (l gormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }

func (l gormLogger) Info(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, fmt.Sprintf(msg, args...))
}

func (l gormLogger) Warn(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, fmt.Sprintf(msg, args...))
}

func (l gormLogger) Error(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, fmt.Sprintf(msg, args...))
}

// ParamsFilter drops the bound values so the logged SQL keeps only $1, $2... placeholders.
// GORM finds it by type assertion on the logger.
func (l gormLogger) ParamsFilter(_ context.Context, sql string, _ ...any) (string, []any) {
	return sql, nil
}

func (l gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, gorm.ErrDuplicatedKey):
		return
	case err != nil:
		sql, rows := fc()
		l.logger.ErrorContext(ctx, "query failed", "error", err, "sql", sql, "rows", rows, "elapsed_ms", ms(elapsed))
	case elapsed > slowQueryThreshold:
		sql, rows := fc()
		l.logger.WarnContext(ctx, "slow query", "sql", sql, "rows", rows, "elapsed_ms", ms(elapsed))
	case l.logger.Enabled(ctx, slog.LevelDebug):
		sql, rows := fc()
		l.logger.DebugContext(ctx, "query", "sql", sql, "rows", rows, "elapsed_ms", ms(elapsed))
	}
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }
