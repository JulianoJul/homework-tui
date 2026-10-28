package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"homework/internal/model"
)

func TestLoadMissingAndSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "data.json")
	s := New(path)
	data, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Subjects) != 0 || data.Version != model.DataVersion {
		t.Fatalf("unexpected new data: %+v", data)
	}

	data.Subjects = append(data.Subjects, model.Subject{ID: "abc", Name: "Álgebra", Folder: "algebra", CreatedAt: time.Now()})
	data.Tasks = append(data.Tasks, model.Task{ID: "t1", SubjectID: "abc", Title: "TP", CreatedAt: time.Now()})
	if err := s.Save(data); err != nil {
		t.Fatal(err)
	}

	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Subjects) != 1 || got.Subjects[0].Name != "Álgebra" {
		t.Fatalf("subjects = %+v", got.Subjects)
	}
	if len(got.Tasks) != 1 || got.Tasks[0].Title != "TP" {
		t.Fatalf("tasks = %+v", got.Tasks)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := osWrite(path, "{broken"); err != nil {
		t.Fatal(err)
	}
	if _, err := New(path).Load(); err == nil {
		t.Fatal("expected error")
	}
}

func osWrite(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
