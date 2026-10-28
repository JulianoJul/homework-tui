package discover

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSubjects(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"algebra", "Fisica", "go", "python", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "algebra", "notas.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go", "go.mod"), []byte("module x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "python", "pyproject.toml"), []byte("[project]"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Subjects(root, []string{"go", "python"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"algebra", "Fisica"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Subjects = %#v, want %#v", got, want)
	}
}

func TestSubjectsMissingRoot(t *testing.T) {
	if _, err := Subjects(filepath.Join(t.TempDir(), "missing"), nil); err == nil {
		t.Fatal("expected error")
	}
}
