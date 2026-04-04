package logging

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want slog.Level
	}{
		{name: "default info", raw: "", want: slog.LevelInfo},
		{name: "debug", raw: "debug", want: slog.LevelDebug},
		{name: "warn", raw: "warn", want: slog.LevelWarn},
		{name: "warning", raw: "warning", want: slog.LevelWarn},
		{name: "error", raw: "error", want: slog.LevelError},
		{name: "trim and case fold", raw: " DEBUG ", want: slog.LevelDebug},
		{name: "unknown falls back to info", raw: "trace", want: slog.LevelInfo},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := parseLevel(tc.raw); got != tc.want {
				t.Fatalf("parseLevel(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
