package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetViper clears all viper state between tests.
func resetViper() {
	viper.Reset()
}

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "test",
		Version: "test-version",
		RunE:    func(cmd *cobra.Command, args []string) error { return nil },
	}
	BindFlags(cmd)
	return cmd
}

func TestDefaultValues(t *testing.T) {
	resetViper()
	cmd := newTestCmd()
	cmd.SetArgs([]string{})
	cmd.Execute()

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Host != DefaultHost {
		t.Errorf("Host = %q, want %q", cfg.Host, DefaultHost)
	}
	if cfg.Port != DefaultPort {
		t.Errorf("Port = %d, want %d", cfg.Port, DefaultPort)
	}
	if cfg.Username != DefaultUsername {
		t.Errorf("Username = %q, want %q", cfg.Username, DefaultUsername)
	}
	if cfg.LogLevel != DefaultLogLevel {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, DefaultLogLevel)
	}
	if cfg.Offline != false {
		t.Errorf("Offline = %v, want false", cfg.Offline)
	}
	if cfg.RCONPort != DefaultRCONPort {
		t.Errorf("RCONPort = %d, want %d", cfg.RCONPort, DefaultRCONPort)
	}
	if cfg.RCONPassword != "" {
		t.Errorf("RCONPassword = %q, want empty", cfg.RCONPassword)
	}
}

func TestCLIFlagsOverrideDefaults(t *testing.T) {
	resetViper()
	cmd := newTestCmd()
	cmd.SetArgs([]string{"--host", "mc.example.com", "--port", "25566", "--username", "TestBot", "--log-level", "debug"})
	cmd.Execute()

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Host != "mc.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "mc.example.com")
	}
	if cfg.Port != 25566 {
		t.Errorf("Port = %d, want %d", cfg.Port, 25566)
	}
	if cfg.Username != "TestBot" {
		t.Errorf("Username = %q, want %q", cfg.Username, "TestBot")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestEnvVarsOverrideConfigFile(t *testing.T) {
	resetViper()

	// Write a config file
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(configFile, []byte("host: filehost\nport: 11111\n"), 0644)

	// Set env var — should beat config file
	t.Setenv("CLANKERCRAFT_HOST", "envhost")

	cmd := newTestCmd()
	cmd.SetArgs([]string{"--config", configFile})
	cmd.Execute()

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Host != "envhost" {
		t.Errorf("Host = %q, want %q (env should beat config file)", cfg.Host, "envhost")
	}
	if cfg.Port != 11111 {
		t.Errorf("Port = %d, want %d (config file should be loaded)", cfg.Port, 11111)
	}
}

func TestCLIFlagsOverrideEnvVars(t *testing.T) {
	resetViper()

	t.Setenv("CLANKERCRAFT_HOST", "envhost")

	cmd := newTestCmd()
	cmd.SetArgs([]string{"--host", "clihost"})
	cmd.Execute()

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Host != "clihost" {
		t.Errorf("Host = %q, want %q (CLI should beat env)", cfg.Host, "clihost")
	}
}

func TestConfigFileLoading(t *testing.T) {
	resetViper()

	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(configFile, []byte("host: fromfile\nport: 22222\nusername: FileBot\n"), 0644)

	cmd := newTestCmd()
	cmd.SetArgs([]string{"--config", configFile})
	cmd.Execute()

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Host != "fromfile" {
		t.Errorf("Host = %q, want %q", cfg.Host, "fromfile")
	}
	if cfg.Port != 22222 {
		t.Errorf("Port = %d, want %d", cfg.Port, 22222)
	}
	if cfg.Username != "FileBot" {
		t.Errorf("Username = %q, want %q", cfg.Username, "FileBot")
	}
}

func TestMissingConfigFileIsNotError(t *testing.T) {
	resetViper()

	cmd := newTestCmd()
	cmd.SetArgs([]string{}) // no --config, no file on disk
	cmd.Execute()

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("Load() should not error with missing config file: %v", err)
	}
	if cfg.Host != DefaultHost {
		t.Errorf("Host = %q, want default %q", cfg.Host, DefaultHost)
	}
}

func TestExplicitMissingConfigFileIsError(t *testing.T) {
	resetViper()

	cmd := newTestCmd()
	cmd.SetArgs([]string{"--config", "/nonexistent/config.yaml"})
	cmd.Execute()

	_, err := Load(cmd)
	if err == nil {
		t.Error("Load() should error when explicit config file doesn't exist")
	}
}

func TestVersionFlag(t *testing.T) {
	resetViper()
	cmd := newTestCmd()
	cmd.SetArgs([]string{"--version"})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--version should not error: %v", err)
	}

	if !strings.Contains(out.String(), "test-version") {
		t.Errorf("--version output = %q, want it to contain %q", out.String(), "test-version")
	}
}

func TestMaskedRCONPassword(t *testing.T) {
	cfg := &Config{RCONPassword: "secret"}
	if got := cfg.MaskedRCONPassword(); got != "***" {
		t.Errorf("MaskedRCONPassword() = %q, want %q", got, "***")
	}

	cfg2 := &Config{RCONPassword: ""}
	if got := cfg2.MaskedRCONPassword(); got != "" {
		t.Errorf("MaskedRCONPassword() = %q, want empty", got)
	}
}

func TestFullLayeringPriority(t *testing.T) {
	resetViper()

	// Layer 1: Config file sets host=filehost
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(configFile, []byte("host: filehost\nusername: FileBot\n"), 0644)

	// Layer 2: Env var sets host=envhost (beats file)
	t.Setenv("CLANKERCRAFT_HOST", "envhost")

	// Layer 3: CLI flag sets host=clihost (beats env)
	cmd := newTestCmd()
	cmd.SetArgs([]string{"--config", configFile, "--host", "clihost"})
	cmd.Execute()

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// CLI wins for host
	if cfg.Host != "clihost" {
		t.Errorf("Host = %q, want %q (CLI > env > file)", cfg.Host, "clihost")
	}
	// Config file should still provide username (no CLI or env override)
	if cfg.Username != "FileBot" {
		t.Errorf("Username = %q, want %q (file should provide unoverridden values)", cfg.Username, "FileBot")
	}
}
