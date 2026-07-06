package logs

import (
	"log/slog"
	"testing"
)

func TestSetup(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "unknown"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			Setup(level, "test-service", "test-env")
			if slog.Default() == nil {
				t.Error("expected slog.Default() to be set")
			}
		})
	}
}

func TestHandler_WithGroup(t *testing.T) {
	inner := slog.NewTextHandler(nil, nil)
	h := NewSanitizingHandler(inner, BlockedFields)
	h2 := h.WithGroup("testgroup")
	if h2 == nil {
		t.Error("WithGroup should return a handler")
	}
}
