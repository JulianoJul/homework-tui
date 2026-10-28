package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"homework/internal/model"
)

// Store persists the whole dataset as one human-readable JSON file. A single
// file makes it trivial to share between the Go and Python versions.
type Store struct {
	Path string
}

func New(path string) *Store {
	return &Store{Path: path}
}

// Load reads the store, creating an empty dataset when the file does not exist.
func (s *Store) Load() (*model.Data, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.NewData(), nil
		}
		return nil, err
	}
	var out model.Data
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("no pude leer %s: %w", s.Path, err)
	}
	return &out, nil
}

// Save writes atomically so a crash or full disk cannot leave half a JSON file.
func (s *Store) Save(data *model.Data) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".data-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.Path); err != nil {
		return err
	}
	return nil
}
