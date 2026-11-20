package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewLoggerSetsLevelByEnvironment(t *testing.T) {
	originalLogger := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	for _, testCase := range []struct {
		environment string
		debug       bool
		info        bool
		warn        bool
	}{
		{environment: "development", debug: true, info: true, warn: true},
		{environment: "production", debug: false, info: false, warn: true},
	} {
		t.Run(testCase.environment, func(t *testing.T) {
			NewLogger("test-service", "", testCase.environment)
			logger := slog.Default()

			if got := logger.Enabled(context.Background(), slog.LevelDebug); got != testCase.debug {
				t.Errorf("debug enabled = %t, want %t", got, testCase.debug)
			}
			if got := logger.Enabled(context.Background(), slog.LevelInfo); got != testCase.info {
				t.Errorf("info enabled = %t, want %t", got, testCase.info)
			}
			if got := logger.Enabled(context.Background(), slog.LevelWarn); got != testCase.warn {
				t.Errorf("warn enabled = %t, want %t", got, testCase.warn)
			}
		})
	}
}
