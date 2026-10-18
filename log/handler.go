package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const (
	ansiReset   = "\033[0m"
	ansiGrey    = "\033[90m"
	ansiRed     = "\033[31m"
	ansiYellow  = "\033[33m"
	ansiGreen   = "\033[32m"
	ansiBlue    = "\033[34m"
	ansiMagenta = "\033[35m"

	timeFormat = "2006-01-02T15:04:05.000000Z07:00"
)

type MultiHandler struct {
	hs []slog.Handler
}

type LevelFilter struct {
	pass func(slog.Level) bool
	h    slog.Handler
}

type colorHandler struct {
	w      io.Writer
	level  slog.Leveler
	prefix string
}

func (m MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.hs {
		if h.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

func (m MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.hs {
		_ = h.Handle(ctx, r)
	}

	return nil
}

func (m MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Handler, len(m.hs))

	for i, h := range m.hs {
		out[i] = h.WithAttrs(attrs)
	}

	return MultiHandler{hs: out}
}

func (m MultiHandler) WithGroup(name string) slog.Handler {
	out := make([]slog.Handler, len(m.hs))

	for i, h := range m.hs {
		out[i] = h.WithGroup(name)
	}

	return MultiHandler{hs: out}
}

func (m MultiHandler) WithoutConsole() MultiHandler {
	var hs []slog.Handler
	for _, h := range m.hs {
		filter, isFilter := h.(LevelFilter)
		if isFilter {
			_, isConsole := filter.h.(*colorHandler)
			if isConsole {
				continue
			}
		}
		hs = append(hs, h)
	}

	return MultiHandler{hs: hs}
}

func (f LevelFilter) Enabled(ctx context.Context, level slog.Level) bool {
	if !f.pass(level) {
		return false
	}

	return f.h.Enabled(ctx, level)
}

func (f LevelFilter) Handle(ctx context.Context, r slog.Record) error {
	if !f.pass(r.Level) {
		return nil
	}

	return f.h.Handle(ctx, r)
}

func (f LevelFilter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return LevelFilter{pass: f.pass, h: f.h.WithAttrs(attrs)}
}

func (f LevelFilter) WithGroup(name string) slog.Handler {
	return LevelFilter{pass: f.pass, h: f.h.WithGroup(name)}
}

func (h *colorHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *colorHandler) Handle(_ context.Context, r slog.Record) error {
	var buf strings.Builder

	buf.WriteString(h.prefix)
	buf.WriteString(ansiGrey)
	buf.WriteString(r.Time.Format(timeFormat))
	buf.WriteString(ansiReset)
	buf.WriteString(" ")

	levelName := r.Level.String()
	if r.Level == LevelTrace {
		levelName = "TRACE"
	}

	buf.WriteString(levelColor(r.Level))
	fmt.Fprintf(&buf, "%5s", levelName)
	buf.WriteString(ansiReset)

	buf.WriteString(" ")
	buf.WriteString(r.Message)

	r.Attrs(func(a slog.Attr) bool {
		buf.WriteString(" ")
		buf.WriteString(ansiGrey)
		buf.WriteString(a.Key)
		buf.WriteString("=")
		buf.WriteString(ansiReset)
		switch v := a.Value.Any().(type) {
		case []any, map[string]any:
			if len(a.Value.String()) > 16 {
				indented, _ := json.MarshalIndent(v, "", "    ")
				buf.Write(indented)
			} else {
				buf.WriteString(a.Value.String())
			}
		default:
			fmt.Fprintf(&buf, "%+v", a.Value.Any())
		}
		return true
	})

	buf.WriteString("\n")

	_, err := io.WriteString(h.w, buf.String())

	return err
}

func (h *colorHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *colorHandler) WithGroup(_ string) slog.Handler {
	return h
}

func levelColor(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return ansiRed

	case level >= slog.LevelWarn:
		return ansiYellow

	case level >= slog.LevelInfo:
		return ansiGreen

	case level >= slog.LevelDebug:
		return ansiBlue

	case level >= LevelTrace:
		return ansiMagenta

	default:
		return ansiReset
	}
}
