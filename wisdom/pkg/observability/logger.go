package observability

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger = slog.Default()

// InitLogger initializes the global structured logger.
// In the future, this will support different handlers (JSON, Text) and levels.
func InitLogger() {
	// Default to JSON for machine-readability in the UI (Neural Atlas)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	Logger = slog.New(handler)
	slog.SetDefault(Logger)

	Logger.Info("Structured logger initialized", "level", "INFO", "format", "JSON")
}
