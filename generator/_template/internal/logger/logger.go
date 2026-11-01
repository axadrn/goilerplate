package logger

import (
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	slogmulti "github.com/samber/slog-multi"
	slogsentry "github.com/samber/slog-sentry/v2"
)

// Errors logs joined errors as separate structured events.
func Errors(message string, err error) {
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		for _, issue := range joined.Unwrap() {
			Errors(message, issue)
		}
		return
	}
	slog.Error(message, "error", err)
}

// Config is the logger's runtime configuration.
type Config struct {
	Level         string
	Format        string
	SentryDSN     string
	IsDevelopment bool
}

// Load reads the environment values owned by the logger.
func Load(isDevelopment bool) Config {
	return Config{
		Level:         strings.TrimSpace(os.Getenv("LOG_LEVEL")),
		Format:        strings.TrimSpace(os.Getenv("LOG_FORMAT")),
		SentryDSN:     strings.TrimSpace(os.Getenv("SENTRY_DSN")),
		IsDevelopment: isDevelopment,
	}
}

// Init installs the process logger.
func Init(cfg Config) {
	level, validLevel := parseLevel(cfg.Level, cfg.IsDevelopment)
	format := strings.ToLower(cfg.Format)
	if format == "" {
		format = "pretty"
	}
	validFormat := format == "pretty" || format == "json"
	if !validFormat {
		format = "pretty"
	}

	var console slog.Handler
	if format == "json" {
		console = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		console = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      level,
			TimeFormat: "15:04:05.000 MST",
			// Drop color when stdout is not a terminal (file, journald, shipper).
			NoColor: !isatty.IsTerminal(os.Stdout.Fd()),
		})
	}

	handlers := []slog.Handler{console}
	var sentryErr error
	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			TracesSampleRate: 1.0,
		}); err != nil {
			sentryErr = err
		} else {
			handlers = append(handlers, slogsentry.Option{Level: slog.LevelError}.NewSentryHandler())
		}
	}

	var handler slog.Handler
	if len(handlers) > 1 {
		handler = slogmulti.Fanout(handlers...)
	} else {
		handler = handlers[0]
	}

	slog.SetDefault(slog.New(handler))
	if !validLevel {
		slog.Warn("invalid LOG_LEVEL, using default", "value", cfg.Level)
	}
	if !validFormat {
		slog.Warn("invalid LOG_FORMAT, using pretty", "value", cfg.Format)
	}
	if sentryErr != nil {
		slog.Warn("sentry initialization failed", "error", sentryErr)
	}
}

func parseLevel(value string, isDevelopment bool) (slog.Level, bool) {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	case "":
		if isDevelopment {
			return slog.LevelDebug, true
		}
		return slog.LevelInfo, true
	}
	if isDevelopment {
		return slog.LevelDebug, false
	}
	return slog.LevelInfo, false
}
