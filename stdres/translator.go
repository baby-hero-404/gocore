package stdres

import "context"

type contextKey string

const (
	translatorContextKey contextKey = "gocore_translator"
)

// Translator defines an optional translation interface for localization (i18n).
type Translator interface {
	Translate(ctx context.Context, key string, args ...any) string
}

// WithTranslator returns a new context containing the provided Translator.
func WithTranslator(ctx context.Context, t Translator) context.Context {
	return context.WithValue(ctx, translatorContextKey, t)
}

// TranslatorFromContext retrieves the Translator from ctx if available.
func TranslatorFromContext(ctx context.Context) Translator {
	if ctx == nil {
		return nil
	}
	if t, ok := ctx.Value(translatorContextKey).(Translator); ok {
		return t
	}
	return nil
}
