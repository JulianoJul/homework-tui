package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpand(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home")
	}
	if got, want := Expand("~/Documentos"), filepath.Join(home, "Documentos"); got != want {
		t.Fatalf("Expand = %q, want %q", got, want)
	}
	if got := Expand("/tmp/x"); got != "/tmp/x" {
		t.Fatalf("absolute path changed: %q", got)
	}
}

func TestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := Default(dir)
	cfg.Openers["PDF"] = []string{"okular"}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root != cfg.Root {
		t.Fatalf("root = %q, want %q", got.Root, cfg.Root)
	}
	command := got.Openers["pdf"]
	if len(command) != 1 || command[0] != "okular" {
		t.Fatalf("openers = %#v", got.Openers)
	}
}

func TestNormalizePaths(t *testing.T) {
	t.Setenv("HOMEWORK_CONFIG", "~/config-test.json")
	if got := ConfigPath(); got == "" || got[0] == '~' {
		t.Fatalf("ConfigPath = %q", got)
	}
	t.Setenv("HOMEWORK_DATA", "/tmp/homework-data.json")
	if got := DataPath(); got != "/tmp/homework-data.json" {
		t.Fatalf("DataPath = %q", got)
	}
}
