package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all resolved configuration values.
type Config struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	LogLevel     string `mapstructure:"log_level"`
	Offline      bool   `mapstructure:"offline"`
	RCONPort     int    `mapstructure:"rcon_port"`
	RCONPassword    string `mapstructure:"rcon_password"`
	AnthropicAPIKey string `mapstructure:"anthropic_api_key"`
	LLMModel        string `mapstructure:"llm_model"`
}

// defaults
const (
	DefaultHost     = "localhost"
	DefaultPort     = 25565
	DefaultUsername = "LLMBot"
	DefaultLogLevel = "info"
	DefaultRCONPort = 25575
)

// BindFlags registers CLI flags on the given cobra command and binds them to viper.
func BindFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.String("host", DefaultHost, "Minecraft server address")
	f.Int("port", DefaultPort, "Minecraft server port")
	f.String("username", DefaultUsername, "Bot in-game username")
	f.String("log-level", DefaultLogLevel, "Log level (debug, info, warn, error)")
	f.Bool("offline", false, "Use offline/cracked server mode")
	f.Int("rcon-port", DefaultRCONPort, "RCON port")
	f.String("rcon-password", "", "RCON password")
	f.String("config", "", "Config file path (default ~/.config/clankercraft/config.yaml)")

	_ = viper.BindPFlag("host", f.Lookup("host"))
	_ = viper.BindPFlag("port", f.Lookup("port"))
	_ = viper.BindPFlag("username", f.Lookup("username"))
	_ = viper.BindPFlag("log_level", f.Lookup("log-level"))
	_ = viper.BindPFlag("offline", f.Lookup("offline"))
	_ = viper.BindPFlag("rcon_port", f.Lookup("rcon-port"))
	_ = viper.BindPFlag("rcon_password", f.Lookup("rcon-password"))
}

// Load resolves configuration from CLI flags, env vars, config file, and defaults.
// Priority: CLI flags > env vars > config file > defaults.
func Load(cmd *cobra.Command) (*Config, error) {
	// Defaults
	viper.SetDefault("host", DefaultHost)
	viper.SetDefault("port", DefaultPort)
	viper.SetDefault("username", DefaultUsername)
	viper.SetDefault("log_level", DefaultLogLevel)
	viper.SetDefault("offline", false)
	viper.SetDefault("rcon_port", DefaultRCONPort)
	viper.SetDefault("rcon_password", "")
	viper.SetDefault("anthropic_api_key", "")
	viper.SetDefault("llm_model", "")

	// Env vars: CLANKERCRAFT_HOST, CLANKERCRAFT_PORT, etc.
	viper.SetEnvPrefix("CLANKERCRAFT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Config file
	configPath, _ := cmd.Flags().GetString("config")
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(filepath.Join(home, ".config", "clankercraft"))
		}
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file exists but is malformed — report regardless of how it was found
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Support standard ANTHROPIC_API_KEY env var (SDK convention) as fallback.
	if cfg.AnthropicAPIKey == "" {
		cfg.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	return &cfg, nil
}

// MaskedRCONPassword returns "***" if password is set, empty string otherwise.
func (c *Config) MaskedRCONPassword() string {
	if c.RCONPassword != "" {
		return "***"
	}
	return ""
}
