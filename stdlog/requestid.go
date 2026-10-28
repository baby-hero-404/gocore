package stdlog

import "context"

// ContextWithRequestID returns a copy of ctx carrying id as the request ID, so WithContext
// attaches it to every log line. Transport adapters (HTTP, gRPC, ...) share this helper.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDCtxKey, id)
}

// RequestIDFromContext returns the request ID stored by ContextWithRequestID, or "".
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(RequestIDCtxKey).(string)
	return id
}
