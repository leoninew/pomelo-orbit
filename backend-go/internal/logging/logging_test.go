package logging

import (
	"log/slog"
	"testing"
)

func TestNewReturnsLogger(t *testing.T) {
	cases := []struct {
		level       string
		enabledInfo bool
	}{
		{level: "DEBUG", enabledInfo: true},
		{level: "INFO", enabledInfo: true},
		{level: "WARN", enabledInfo: false},
		{level: "ERROR", enabledInfo: false},
		{level: "unknown", enabledInfo: true},
	}
	for _, tc := range cases {
		logger := New(tc.level)
		if logger == nil {
			t.Fatalf("expected logger for level %s", tc.level)
		}
		if logger.Enabled(t.Context(), slog.LevelInfo) != tc.enabledInfo {
			t.Fatalf("unexpected info enabled for level %s", tc.level)
		}
	}
}
