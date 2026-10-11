package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

const (
	unmatchedRoute = "unmatched"
	healthRoute    = "/health"
)

// RequestLog logs one line per request. The line doubles as a CloudWatch
// Embedded Metric Format record, which CloudWatch only parses from JSON logs.
// It logs the route pattern only, never the raw path, query or body.
func RequestLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = unmatchedRoute
			}
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			statusClass := fmt.Sprintf("%dxx", status/100)

			level := slog.LevelInfo
			switch {
			case status >= http.StatusInternalServerError:
				level = slog.LevelError
			case route == healthRoute:
				level = slog.LevelDebug
			}

			logger.LogAttrs(r.Context(), level, "request",
				slog.String("request_id", chimw.GetReqID(r.Context())),
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", status),
				slog.String("status_class", statusClass),
				slog.Int("requests", 1),
				slog.Float64("latency_ms", float64(time.Since(start).Microseconds())/1000),
				slog.Any("_aws", metricsDirective()),
			)
		})
	}
}

func metricsDirective() map[string]any {
	return map[string]any{
		"Timestamp": time.Now().UnixMilli(),
		"CloudWatchMetrics": []map[string]any{{
			"Namespace":  "CardsAPI",
			"Dimensions": [][]string{{"method", "route", "status_class"}},
			"Metrics": []map[string]string{
				{"Name": "requests", "Unit": "Count"},
				{"Name": "latency_ms", "Unit": "Milliseconds"},
			},
		}},
	}
}
