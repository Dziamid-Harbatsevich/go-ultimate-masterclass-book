package platform

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// AssignCorrelationID pulls or builds an X-Request-ID header sequence
func AssignCorrelationID(r *http.Request) string {
	id := r.Header.Get("X-Request-ID")
	if id != "" {
		return id
	}
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetRequestID safely extracts the correlation key from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return "unknown"
}
