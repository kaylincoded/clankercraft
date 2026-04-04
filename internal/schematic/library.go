package schematic

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SchematicInfo holds metadata about an indexed schematic file.
type SchematicInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

// Library indexes .schem files from a directory.
type Library struct {
	dir        string
	logger     *slog.Logger
	schematics map[string]SchematicInfo
}

// NewLibrary creates a Library that will scan the given directory.
func NewLibrary(dir string, logger *slog.Logger) *Library {
	return &Library{
		dir:        dir,
		logger:     logger,
		schematics: make(map[string]SchematicInfo),
	}
}

// Load ensures the schematics directory exists and indexes all .schem files.
func (l *Library) Load() error {
	if err := os.MkdirAll(l.dir, 0755); err != nil {
		return fmt.Errorf("creating schematics directory: %w", err)
	}

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return fmt.Errorf("reading schematics directory: %w", err)
	}

	l.schematics = make(map[string]SchematicInfo)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".schem") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			l.logger.Warn("skipping schematic", slog.String("file", entry.Name()), slog.Any("error", err))
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".schem")
		l.schematics[name] = SchematicInfo{
			Name:      name,
			Path:      filepath.Join(l.dir, entry.Name()),
			SizeBytes: info.Size(),
		}
	}

	return nil
}

// List returns all indexed schematics sorted by name.
func (l *Library) List() []SchematicInfo {
	result := make([]SchematicInfo, 0, len(l.schematics))
	for _, s := range l.schematics {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Has reports whether a schematic with the given name exists.
func (l *Library) Has(name string) bool {
	_, ok := l.schematics[name]
	return ok
}

// Path returns the full filesystem path for a schematic name.
func (l *Library) Path(name string) (string, error) {
	s, ok := l.schematics[name]
	if !ok {
		return "", fmt.Errorf("schematic not found: %q", name)
	}
	return s.Path, nil
}

// Len returns the number of indexed schematics.
func (l *Library) Len() int {
	return len(l.schematics)
}
