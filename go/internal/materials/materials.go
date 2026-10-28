// Package materials scans subject folders and hands files to the desktop's
// default applications.
package materials

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	MaxEntries = 4000
	maxDepth   = 12
)

type Entry struct {
	Path    string
	Rel     string
	Name    string
	Ext     string
	Size    int64
	ModTime time.Time
}

// Summary describes the folder layout, not the student's actual progress.
type Summary struct {
	Activities []string
	Created    int
	Other      int
}

// Summarize includes empty numbered folders and counts files under any
// created_by_me directory separately from all other files.
func Summarize(root string, entries []Entry) (Summary, error) {
	var summary Summary
	dirs, err := os.ReadDir(root)
	if err != nil {
		return summary, err
	}
	for _, dir := range dirs {
		name := dir.Name()
		if !dir.IsDir() || name == "" || strings.Trim(name, "0123456789") != "" {
			continue
		}
		summary.Activities = append(summary.Activities, name)
	}
	sort.Slice(summary.Activities, func(i, j int) bool {
		a, errA := strconv.ParseUint(summary.Activities[i], 10, 64)
		b, errB := strconv.ParseUint(summary.Activities[j], 10, 64)
		if errA == nil && errB == nil && a != b {
			return a < b
		}
		return summary.Activities[i] < summary.Activities[j]
	})
	for _, entry := range entries {
		created := false
		parts := strings.Split(entry.Rel, "/")
		for _, part := range parts[:len(parts)-1] {
			if strings.EqualFold(part, "created_by_me") {
				created = true
				break
			}
		}
		if created {
			summary.Created++
		} else {
			summary.Other++
		}
	}
	return summary, nil
}

// Scan walks a subject folder recursively, skipping hidden files and obvious
// temporary files. It returns a sorted list and whether the safety limit was
// reached.
func Scan(root string) ([]Entry, bool, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, false, fmt.Errorf("no puedo abrir la carpeta: %w", err)
	}
	if !info.IsDir() {
		return nil, false, fmt.Errorf("no es una carpeta: %s", root)
	}

	entries := make([]Entry, 0, 64)
	truncated := false
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("no puedo leer %s: %w", path, walkErr)
		}
		if path == root {
			return nil
		}
		name := d.Name()
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if strings.Count(rel, string(filepath.Separator)) >= maxDepth {
				return fs.SkipDir
			}
			switch strings.ToLower(name) {
			case "node_modules", "__pycache__", ".venv", "venv":
				return fs.SkipDir
			}
			return nil
		}
		if len(entries) >= MaxEntries {
			truncated = true
			return fs.SkipAll
		}
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("no puedo leer %s: %w", path, err)
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
		entries = append(entries, Entry{
			Path:    path,
			Rel:     filepath.ToSlash(rel),
			Name:    name,
			Ext:     ext,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := strings.ToLower(entries[i].Rel), strings.ToLower(entries[j].Rel)
		if a == b {
			return entries[i].Rel < entries[j].Rel
		}
		return a < b
	})
	return entries, truncated, nil
}

// Open launches the configured opener (or xdg-open/open/start) without
// blocking the TUI. Openers are intended for GUI programs.
func Open(path string, openers map[string][]string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if !info.IsDir() {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
		if parts, ok := openers[ext]; ok && len(parts) > 0 {
			args := append(append([]string(nil), parts[1:]...), path)
			cmd = exec.Command(parts[0], args...)
		}
	}
	if cmd == nil {
		cmd = systemOpen(path, info.IsDir())
	}
	if cmd == nil {
		return fmt.Errorf("no sé cómo abrir archivos en %s", runtime.GOOS)
	}
	if _, err := exec.LookPath(cmd.Path); err != nil {
		return fmt.Errorf("no encontré el programa %q", cmd.Path)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func systemOpen(path string, isDir bool) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path)
	case "windows":
		if isDir {
			return exec.Command("explorer", path)
		}
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		return exec.Command("xdg-open", path)
	}
}

func HumanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit && exp < 4 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTP"[exp])
}
