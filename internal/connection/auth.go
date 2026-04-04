package connection

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Tnze/go-mc/bot"
	msauth "github.com/maxsupermanhd/go-mc-ms-auth"
	"github.com/kaylincoded/clankercraft/internal/config"
)

// DefaultClientID is the Azure AD client ID used for MSA device code flow.
// Custom client IDs require Mojang whitelisting.
const DefaultClientID = "88650e7e-efee-4857-b9a9-cf580a00ef43"

// DefaultCacheDir is the directory name for token cache under the user's config dir.
const DefaultCacheDir = "tokens"

// DefaultCacheFile is the filename for the MSA token cache.
const DefaultCacheFile = "msa-cache.json"

// Authenticate performs MSA device code authentication and returns a populated bot.Auth.
// On first run, the user is prompted with a device code and verification URL.
// On subsequent runs, cached tokens are refreshed automatically.
func Authenticate(cfg *config.Config, logger *slog.Logger) (*bot.Auth, error) {
	cachePath, err := ensureCacheDir()
	if err != nil {
		return nil, fmt.Errorf("creating token cache directory: %w", err)
	}

	cacheFile := filepath.Join(cachePath, DefaultCacheFile)

	logger.Info("authenticating with Microsoft account",
		slog.String("cache", cacheFile),
	)

	// GetMCcredentials handles the full flow:
	// 1. Check cache → refresh if expired → or device code flow if no cache
	// 2. MSA → XBL → XSTS → MC token → MC profile
	// The library logs device code prompts via log.Print (goes to stderr)
	creds, err := msauth.GetMCcredentials(cacheFile, DefaultClientID)
	if err != nil {
		return nil, fmt.Errorf("MSA authentication failed: %w", err)
	}

	logger.Info("authenticated successfully",
		slog.String("username", creds.Name),
	)

	return &bot.Auth{
		Name: creds.Name,
		UUID: creds.UUID,
		AsTk: creds.AsTk,
	}, nil
}

// ensureCacheDir creates the token cache directory with restricted permissions.
// Returns the path to the cache directory.
func ensureCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	dir := filepath.Join(home, ".config", "clankercraft", DefaultCacheDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return "", fmt.Errorf("setting permissions on %s: %w", dir, err)
	}

	return dir, nil
}
