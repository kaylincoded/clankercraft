package connection

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureCacheDirCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir, err := ensureCacheDir()
	if err != nil {
		t.Fatalf("ensureCacheDir() returned error: %v", err)
	}

	expected := filepath.Join(tmpDir, ".config", "clankercraft", DefaultCacheDir)
	if dir != expected {
		t.Errorf("ensureCacheDir() = %q, want %q", dir, expected)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("cache directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("cache path is not a directory")
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("directory permissions = %o, want 0700", info.Mode().Perm())
	}
}

func TestEnsureCacheDirIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Call twice — should not error
	dir1, err := ensureCacheDir()
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	dir2, err := ensureCacheDir()
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if dir1 != dir2 {
		t.Errorf("paths differ: %q vs %q", dir1, dir2)
	}
}

func TestEnsureCacheDirFixesPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Pre-create with wrong permissions
	dir := filepath.Join(tmpDir, ".config", "clankercraft", DefaultCacheDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := ensureCacheDir()
	if err != nil {
		t.Fatalf("ensureCacheDir() returned error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("directory permissions = %o, want 0700 (should fix existing dir)", info.Mode().Perm())
	}
}

func TestDefaultConstants(t *testing.T) {
	if DefaultClientID == "" {
		t.Error("DefaultClientID should not be empty")
	}
	if DefaultCacheDir == "" {
		t.Error("DefaultCacheDir should not be empty")
	}
	if DefaultCacheFile == "" {
		t.Error("DefaultCacheFile should not be empty")
	}
}
