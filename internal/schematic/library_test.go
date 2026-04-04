package schematic

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestLoadWithSchemFiles(t *testing.T) {
	dir := t.TempDir()
	// Create .schem files with some content.
	for _, name := range []string{"castle.schem", "bridge.schem", "tower.schem"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fake-schem-data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	lib := NewLibrary(dir, testLogger())
	if err := lib.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if lib.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", lib.Len())
	}
}

func TestLoadCreatesDirectoryWhenMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent", "schematics")

	lib := NewLibrary(dir, testLogger())
	if err := lib.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected directory")
	}

	if lib.Len() != 0 {
		t.Fatalf("Len() = %d, want 0 for empty dir", lib.Len())
	}
}

func TestListReturnsSorted(t *testing.T) {
	dir := t.TempDir()
	names := []string{"zebra.schem", "alpha.schem", "middle.schem"}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	lib := NewLibrary(dir, testLogger())
	if err := lib.Load(); err != nil {
		t.Fatal(err)
	}

	list := lib.List()
	if len(list) != 3 {
		t.Fatalf("List() len = %d, want 3", len(list))
	}
	if list[0].Name != "alpha" || list[1].Name != "middle" || list[2].Name != "zebra" {
		t.Errorf("List() not sorted: got %v, %v, %v", list[0].Name, list[1].Name, list[2].Name)
	}
}

func TestHas(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "castle.schem"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	lib := NewLibrary(dir, testLogger())
	if err := lib.Load(); err != nil {
		t.Fatal(err)
	}

	if !lib.Has("castle") {
		t.Error("Has(castle) = false, want true")
	}
	if lib.Has("nonexistent") {
		t.Error("Has(nonexistent) = true, want false")
	}
}

func TestPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bridge.schem"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	lib := NewLibrary(dir, testLogger())
	if err := lib.Load(); err != nil {
		t.Fatal(err)
	}

	p, err := lib.Path("bridge")
	if err != nil {
		t.Fatalf("Path(bridge) error: %v", err)
	}
	want := filepath.Join(dir, "bridge.schem")
	if p != want {
		t.Errorf("Path(bridge) = %q, want %q", p, want)
	}

	_, err = lib.Path("missing")
	if err == nil {
		t.Fatal("Path(missing) expected error")
	}
}

func TestNonSchemFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	files := []string{"castle.schem", "readme.txt", "backup.schem.bak", "notes.md"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	lib := NewLibrary(dir, testLogger())
	if err := lib.Load(); err != nil {
		t.Fatal(err)
	}

	if lib.Len() != 1 {
		t.Fatalf("Len() = %d, want 1 (only .schem files)", lib.Len())
	}
	if !lib.Has("castle") {
		t.Error("expected castle schematic to be indexed")
	}
}
