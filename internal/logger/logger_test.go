package logger

import (
	"log/slog"
	"testing"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
)

// TestParseLevel 用于单元测试。
func TestParseLevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want slog.Level
	}{
		{in: "debug", want: slog.LevelDebug},
		{in: "warn", want: slog.LevelWarn},
		{in: "error", want: slog.LevelError},
		{in: "info", want: slog.LevelInfo},
		{in: "unknown", want: slog.LevelInfo},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := parseLevel(tc.in); got != tc.want {
				t.Fatalf("parseLevel(%q)=%v want=%v", tc.in, got, tc.want)
			}
		})
	}
}

// TestNewLogger 用于单元测试。
func TestNewLogger(t *testing.T) {
	t.Parallel()

	textLogger := New(appcfg.LogConfig{Format: "text", Level: "info"})
	if textLogger == nil {
		t.Fatalf("text logger should not be nil")
	}
	jsonLogger := New(appcfg.LogConfig{Format: "json", Level: "debug"})
	if jsonLogger == nil {
		t.Fatalf("json logger should not be nil")
	}
}
