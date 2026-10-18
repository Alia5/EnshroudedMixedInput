package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const LevelTrace slog.Level = -8

const logDirMode = 0o755

const ExtensionConsolePrefix = "[EnshroudedMixedInput] "

func Trace(msg string, args ...any) {
	slog.Log(context.Background(), LevelTrace, msg, args...)
}

type TraceLogger struct {
	*slog.Logger
}

func (t *TraceLogger) Trace(msg string, args ...any) {
	t.Log(context.Background(), LevelTrace, msg, args...)
}

func TraceLoggerWith(args ...any) *TraceLogger {
	return &TraceLogger{
		Logger: slog.With(args...),
	}
}

func ParseLevel(name string) slog.Level {
	switch name {
	case "trace":
		return LevelTrace

	case "debug":
		return slog.LevelDebug

	case "info", "":
		return slog.LevelInfo

	case "warn":
		return slog.LevelWarn

	case "error":
		return slog.LevelError

	default:
		return slog.LevelInfo
	}
}

var level slog.LevelVar

func SetLevel(levelName string) {
	level.Set(ParseLevel(levelName))
}

func SetupLogger(levelName, logFile, consolePrefix string, extra ...slog.Handler) (*slog.Logger, []io.Closer, error) {
	SetLevel(levelName)

	var (
		handlers []slog.Handler
		closers  []io.Closer
	)

	handlers = append(handlers, extra...)

	handlers = append(handlers, LevelFilter{
		pass: func(l slog.Level) bool {
			return l < slog.LevelError
		},
		h: &colorHandler{
			w:      os.Stdout,
			level:  &level,
			prefix: consolePrefix,
		},
	})

	handlers = append(handlers, LevelFilter{
		pass: func(l slog.Level) bool {
			return l >= slog.LevelError
		},
		h: &colorHandler{
			w:      os.Stderr,
			level:  slog.LevelError,
			prefix: consolePrefix,
		},
	})

	if logFile != "" {
		err := os.MkdirAll(filepath.Dir(logFile), logDirMode)
		if err != nil {
			return nil, nil, err
		}

		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}

		closers = append(closers, f)
		handlers = append(handlers, slog.NewTextHandler(f, &slog.HandlerOptions{
			Level: &level,
		}))
	}

	logger := slog.New(MultiHandler{hs: handlers})
	slog.SetDefault(logger)

	return logger, closers, nil
}
