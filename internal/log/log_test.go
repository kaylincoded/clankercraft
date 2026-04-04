package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestSetupInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	logger.Info("test message", slog.String("key", "value"))

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}

	if entry["msg"] != "test message" {
		t.Errorf("msg = %v, want %q", entry["msg"], "test message")
	}
	if entry["level"] != "INFO" {
		t.Errorf("level = %v, want %q", entry["level"], "INFO")
	}
	if entry["key"] != "value" {
		t.Errorf("key = %v, want %q", entry["key"], "value")
	}
}

func TestSetupDebugLevel(t *testing.T) {
	logger := Setup("debug")
	if logger == nil {
		t.Fatal("Setup returned nil")
	}
	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug level should be enabled when log level is debug")
	}
}

func TestSetupInfoLevelSuppressesDebug(t *testing.T) {
	logger := Setup("info")
	if logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug level should NOT be enabled when log level is info")
	}
	if !logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("info level should be enabled when log level is info")
	}
}

func TestSetupWarnLevel(t *testing.T) {
	logger := Setup("warn")
	if logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("info level should NOT be enabled when log level is warn")
	}
	if !logger.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("warn level should be enabled when log level is warn")
	}
}

func TestSetupErrorLevel(t *testing.T) {
	logger := Setup("error")
	if logger.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("warn level should NOT be enabled when log level is error")
	}
	if !logger.Enabled(context.Background(), slog.LevelError) {
		t.Error("error level should be enabled when log level is error")
	}
}

func TestSetupDefaultsToInfo(t *testing.T) {
	logger := Setup("unknown")
	if !logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("unknown level should default to info")
	}
	if logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug should not be enabled at default info level")
	}
}
