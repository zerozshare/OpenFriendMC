/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 */
package ipc

import (
	"context"
	"log/slog"
)

type LogHandler struct {
	writer *Writer
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

func NewLogHandler(w *Writer, level slog.Level) *LogHandler {
	return &LogHandler{writer: w, level: level}
}

func (h *LogHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level
}

func (h *LogHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := make(map[string]any, r.NumAttrs()+len(h.attrs))
	for _, a := range h.attrs {
		attrs[a.Key] = jsonSafe(a.Value.Any())
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = jsonSafe(a.Value.Any())
		return true
	})
	payload := map[string]any{
		"level": r.Level.String(),
		"msg":   r.Message,
	}
	if len(attrs) > 0 {
		payload["attrs"] = attrs
	}
	return h.writer.Notify("log", payload)
}

func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &next
}

func (h *LogHandler) WithGroup(name string) slog.Handler {
	next := *h
	next.groups = append(append([]string{}, h.groups...), name)
	return &next
}

func jsonSafe(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case error:
		return x.Error()
	case interface{ String() string }:
		switch v.(type) {
		case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string:
			return v
		}
		return x.String()
	default:
		return v
	}
}
