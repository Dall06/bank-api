package logs

import (
	"context"
	"log/slog"
)

// SanitizingHandler wraps an slog.Handler and masks values whose keys match
// any of the blocked field substrings (case-insensitive).
type SanitizingHandler struct {
	inner         slog.Handler
	blockedFields []string
}

// NewSanitizingHandler returns a handler that sanitizes attribute values before
// delegating to inner.
func NewSanitizingHandler(inner slog.Handler, blockedFields []string) *SanitizingHandler {
	return &SanitizingHandler{
		inner:         inner,
		blockedFields: blockedFields,
	}
}

func (h *SanitizingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *SanitizingHandler) Handle(ctx context.Context, r slog.Record) error {
	sanitized := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		sanitized.AddAttrs(h.sanitizeAttr(a))
		return true
	})
	return h.inner.Handle(ctx, sanitized)
}

func (h *SanitizingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	sanitized := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		sanitized[i] = h.sanitizeAttr(a)
	}
	return NewSanitizingHandler(h.inner.WithAttrs(sanitized), h.blockedFields)
}

func (h *SanitizingHandler) WithGroup(name string) slog.Handler {
	return NewSanitizingHandler(h.inner.WithGroup(name), h.blockedFields)
}

// sanitizeAttr masks blocked keys and recurses into groups.
func (h *SanitizingHandler) sanitizeAttr(a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		sanitized := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			sanitized[i] = h.sanitizeAttr(ga)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(sanitized...)}
	}

	if isBlocked(a.Key, h.blockedFields) {
		return slog.String(a.Key, "****")
	}

	return a
}
