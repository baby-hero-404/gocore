package stdlog

import (
	"errors"
	"log/slog"

	"github.com/baby-hero-404/gocore/stderr"
)

// LevelFor returns an appropriate slog.Level based on the error's classified Code.
// 5xxx (Internal/Unavailable/Timeout/DB/Cache) -> LevelError
// 4xxx (Unauthorized/Forbidden/NotFound/Conflict) -> LevelWarn
// 1xxx (Validation/BadRequest) -> LevelWarn
// Nil or OK -> LevelInfo
func LevelFor(err error) slog.Level {
	if err == nil {
		return slog.LevelInfo
	}

	var stdErr *stderr.Error
	if errors.As(err, &stdErr) {
		switch {
		case stdErr.Code == stderr.CodeOK:
			return slog.LevelInfo
		case stdErr.Code >= 5000:
			return slog.LevelError
		case stdErr.Code >= 4000 && stdErr.Code < 5000:
			return slog.LevelWarn
		case stdErr.Code >= 1000 && stdErr.Code < 2000:
			return slog.LevelWarn
		default:
			return slog.LevelError
		}
	}

	return slog.LevelError
}
