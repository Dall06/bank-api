package logs

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestSanitizingHandler(t *testing.T) {
	tests := []struct {
		name       string
		blocked    []string
		logFunc    func(l *slog.Logger)
		wantMasked []string // keys that should be "****"
		wantClear  []string // keys that should NOT be masked
	}{
		{
			name:    "masks password at top level",
			blocked: []string{"password"},
			logFunc: func(l *slog.Logger) {
				l.Info("test", "username", "john", "password", "secret123")
			},
			wantMasked: []string{"password"},
			wantClear:  []string{"username"},
		},
		{
			name:    "masks multiple blocked fields",
			blocked: []string{"password", "token"},
			logFunc: func(l *slog.Logger) {
				l.Info("test", "password", "secret", "token", "abc", "email", "a@b.com")
			},
			wantMasked: []string{"password", "token"},
			wantClear:  []string{"email"},
		},
		{
			name:    "case insensitive matching",
			blocked: []string{"password"},
			logFunc: func(l *slog.Logger) {
				l.Info("test", "Password", "secret1", "PASSWORD", "secret2")
			},
			wantMasked: []string{"Password", "PASSWORD"},
		},
		{
			name:    "substring matching",
			blocked: []string{"password"},
			logFunc: func(l *slog.Logger) {
				l.Info("test", "user_password", "secret", "password_hash", "hash")
			},
			wantMasked: []string{"user_password", "password_hash"},
		},
		{
			name:    "masks inside groups",
			blocked: []string{"secret"},
			logFunc: func(l *slog.Logger) {
				l.Info("test", slog.Group("auth", slog.String("secret", "hidden"), slog.String("user", "visible")))
			},
			wantMasked: []string{"secret"},
			wantClear:  []string{"user"},
		},
		{
			name:    "no blocked fields passes through",
			blocked: []string{"password"},
			logFunc: func(l *slog.Logger) {
				l.Info("test", "name", "john", "age", 30)
			},
			wantClear: []string{"name"},
		},
		{
			name:    "empty blocked list passes everything",
			blocked: []string{},
			logFunc: func(l *slog.Logger) {
				l.Info("test", "password", "visible")
			},
			wantClear: []string{"password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			base := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			handler := NewSanitizingHandler(base, tt.blocked)
			logger := slog.New(handler)

			tt.logFunc(logger)

			var out map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
				t.Fatalf("failed to parse log JSON: %v\nraw: %s", err, buf.String())
			}

			for _, key := range tt.wantMasked {
				val := findValue(out, key)
				if val != "****" {
					t.Errorf("expected key %q to be masked, got %q", key, val)
				}
			}
			for _, key := range tt.wantClear {
				val := findValue(out, key)
				if val == "****" {
					t.Errorf("expected key %q to NOT be masked, got ****", key)
				}
			}
		})
	}
}

func TestSanitizingHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewJSONHandler(&buf, nil)
	handler := NewSanitizingHandler(base, []string{"token"})
	logger := slog.New(handler).With("token", "secret-token", "service", "test")

	logger.Info("check")

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if out["token"] != "****" {
		t.Errorf("WithAttrs token not masked: %v", out["token"])
	}
	if out["service"] != "test" {
		t.Errorf("WithAttrs service wrong: %v", out["service"])
	}
}

func TestSanitizingHandler_Enabled(t *testing.T) {
	base := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn})
	handler := NewSanitizingHandler(base, nil)

	if handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected Info to be disabled when level is Warn")
	}
	if !handler.Enabled(context.Background(), slog.LevelError) {
		t.Error("expected Error to be enabled when level is Warn")
	}
}

func TestIsBlocked(t *testing.T) {
	tests := []struct {
		key     string
		blocked []string
		want    bool
	}{
		{"password", []string{"password"}, true},
		{"PASSWORD", []string{"password"}, true},
		{"user_password", []string{"password"}, true},
		{"username", []string{"password"}, false},
		{"token", []string{"password", "token"}, true},
		{"name", []string{}, false},
	}
	for _, tt := range tests {
		got := isBlocked(tt.key, tt.blocked)
		if got != tt.want {
			t.Errorf("isBlocked(%q, %v) = %v, want %v", tt.key, tt.blocked, got, tt.want)
		}
	}
}

// findValue searches for a key in a potentially nested JSON map.
func findValue(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	// Search nested maps (for slog groups)
	for _, v := range m {
		if nested, ok := v.(map[string]interface{}); ok {
			if result := findValue(nested, key); result != "" {
				return result
			}
		}
	}
	return ""
}
