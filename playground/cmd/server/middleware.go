package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"playground/internal/platform"
	"runtime/debug"
	"time"
)

// TraceMiddleware injects tracking context structures across handlers
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := platform.AssignCorrelationID(r)
		ctx := context.WithValue(r.Context(), platform.RequestIDKey, reqID)

		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// StructuredLoggingMiddleware pipes runtime request diagnostics directly to NDJSON
func StructuredLoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := platform.GetRequestID(r.Context())

		logger.Info("HTTP request incoming",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("request_id", reqID),
		)

		next.ServeHTTP(w, r)

		logger.Info("HTTP request processed successfully",
			slog.String("path", r.URL.Path),
			slog.String("request_id", reqID),
			slog.Duration("latency", time.Since(start)),
		)
	})
}

// RecoveryMiddleware intercepts catastrophic pointer failures safely
func RecoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				reqID := platform.GetRequestID(r.Context())

				logger.Error("Catastrophic runtime panic intercepted",
					slog.Any("error", err),
					slog.String("request_id", reqID),
					slog.String("stack_trace", string(debug.Stack())),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error":"Internal system operational error occurred","request_id":"%s"}`, reqID)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
