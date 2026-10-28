// Package discover finds subject folders under the materials root.
package discover

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Subjects returns the immediate subdirectories of root that look like
// subjects. Hidden folders, ignored names and source-code checkouts are
// skipped so the Go/Python versions of this app can live inside the same tree
// without showing up as courses.
func Subjects(root string, ignore []string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("no puedo abrir la carpeta de materiales %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("la raíz de materiales no es una carpeta: %s", root)
	}

	ignored := make(map[string]bool, len(ignore))
	for _, name := range ignore {
		ignored[strings.ToLower(strings.TrimSpace(name))] = true
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	folders := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || !entry.IsDir() {
			continue
		}
		if ignored[strings.ToLower(name)] {
			continue
		}
		path := filepath.Join(root, name)
		if isSourceTree(path) {
			continue
		}
		folders = append(folders, name)
	}
	sort.Slice(folders, func(i, j int) bool {
		return strings.ToLower(folders[i]) < strings.ToLower(folders[j])
	})
	return folders, nil
}

func isSourceTree(dir string) bool {
	for _, marker := range []string{"go.mod", "pyproject.toml", "Cargo.toml", ".git"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}
