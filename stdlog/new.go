package stdlog

import (
	"log/slog"
	"os"
	"sync/atomic"
)

// defaultLogger is the process-wide logger behind the package-level helpers.
// It is created lazily on first use (no init side effects) and replaced via SetDefault.
var defaultLogger atomic.Pointer[Logger]

type loggerImpl struct {
	sl *slog.Logger
}

// New creates a new Logger adhering to the Logger interface.
func New(opts ...Option) Logger {
	o := &options{
		level:     slog.LevelInfo,
		isJSON:    true,
		addSource: false,
		output:    os.Stdout,
	}
	for _, opt := range opts {
		opt(o)
	}

	var handler slog.Handler
	if o.handler != nil {
		handler = o.handler
	} else {
		handlerOpts := &slog.HandlerOptions{
			Level:       o.level,
			AddSource:   o.addSource,
			ReplaceAttr: SanitizeAttr,
		}
		if o.isJSON {
			handler = slog.NewJSONHandler(o.output, handlerOpts)
		} else {
			handler = slog.NewTextHandler(o.output, handlerOpts)
		}
	}

	// Always wrap with RedactingHandler to ensure sensitive data is not leaked
	redacting := NewRedactingHandler(handler)

	return &loggerImpl{
		sl: slog.New(redacting),
	}
}

// Get returns the global default logger instance.
func Get() Logger {
	if l := defaultLogger.Load(); l != nil {
		return *l
	}
	l := New()
	if defaultLogger.CompareAndSwap(nil, &l) {
		return l
	}
	return *defaultLogger.Load()
}

// SetDefault replaces the global default logger.
func SetDefault(l Logger) {
	if l != nil {
		defaultLogger.Store(&l)
		slog.SetDefault(l.Slog())
	}
}
