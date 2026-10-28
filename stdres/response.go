package stdres

import (
	"context"
	"errors"
	"net/http"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
)

// Envelope is the standard JSON response envelope for API endpoints.
type Envelope struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Key       string `json:"key,omitempty"`  // i18n key; clients translate by Code/Key
	Args      []any  `json:"args,omitempty"` // i18n arguments for Key
	Data      any    `json:"data,omitempty"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"` // trace id, same as the X-Request-ID header
}

// Success creates an Envelope with Code 0 and "success" message.
func Success(data any) Envelope {
	return Envelope{
		Code:    int(stderr.CodeOK),
		Message: "success",
		Data:    data,
	}
}

// SuccessCtx is Success plus the request ID found in ctx, so success responses carry the same
// trace id as error responses and the X-Request-ID header.
func SuccessCtx(ctx context.Context, data any) Envelope {
	env := Success(data)
	env.RequestID = requestIDFromContext(ctx)
	return env
}

// FromError transforms any error into a standardized Envelope and corresponding HTTP status code.
// If err is a *stderr.Error, its code, status, message, and details are extracted.
// If a translator is available (via parameter or ctx), error message keys are translated.
func FromError(ctx context.Context, err error, translator ...Translator) (Envelope, int) {
	if err == nil {
		return Success(nil), http.StatusOK
	}

	var tr Translator
	if len(translator) > 0 && translator[0] != nil {
		tr = translator[0]
	} else if ctx != nil {
		tr = TranslatorFromContext(ctx)
	}

	var stdErr *stderr.Error
	if errors.As(err, &stdErr) {
		message := stdErr.Message
		if tr != nil && stdErr.Key != "" {
			translated := tr.Translate(ctx, stdErr.Key, stdErr.Args...)
			if translated != "" {
				message = translated
			}
		}

		var details any
		if len(stdErr.Details) > 0 {
			details = stdErr.Details
		}

		return Envelope{
			Code:      int(stdErr.Code),
			Message:   message,
			Key:       stdErr.Key,
			Args:      stdErr.Args,
			Details:   details,
			RequestID: requestIDFromContext(ctx),
		}, stdErr.HTTPStatus()
	}

	// Unknown / plain Go error: map to 500 Internal Server Error. The raw text may contain driver
	// messages, hostnames or DSNs, so it is never sent to the client; transports log it instead.
	return Envelope{
		Code:      int(stderr.CodeInternal),
		Message:   "internal server error",
		RequestID: requestIDFromContext(ctx),
	}, http.StatusInternalServerError
}

// LogUnclassified logs err when it is not a *stderr.Error. FromError hides such errors from the
// client behind a generic message, so this is the only place their text surfaces. Transport
// adapters call it before responding.
func LogUnclassified(ctx context.Context, err error) {
	var stdErr *stderr.Error
	if err == nil || errors.As(err, &stdErr) {
		return
	}
	stdlog.WithContext(ctx).Error("unclassified error returned to client as internal server error", "error", err.Error())
}

func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(stdlog.RequestIDCtxKey).(string)
	return id
}
