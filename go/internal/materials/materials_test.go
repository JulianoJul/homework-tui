package materials

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScan(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("Caso 3.pdf")
	mustWrite("created_by_me/informe.pdf")
	mustWrite(".oculto.txt")
	mustWrite("sub/.oculto")
	if err := os.MkdirAll(filepath.Join(root, "vacia"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries, truncated, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Fatal("unexpected truncation")
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want 2", entries)
	}
	if entries[0].Rel != "Caso 3.pdf" || entries[0].Ext != "pdf" {
		t.Fatalf("first entry = %+v", entries[0])
	}
	if entries[1].Rel != "created_by_me/informe.pdf" {
		t.Fatalf("second entry = %+v", entries[1])
	}
}

func TestScanMissing(t *testing.T) {
	if _, _, err := Scan(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error")
	}
}

func TestSummarizeActivitiesAndCreatedFiles(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"1/created_by_me", "2", "10", "created_by_me", "not-a-number"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"1/created_by_me/mio.pdf", "created_by_me/notas.md", "1/apoyo.pdf", "2/otro.pdf"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	entries, _, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := Summarize(root, entries)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(summary.Activities, ",") != "1,2,10" || summary.Created != 2 || summary.Other != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestHumanSize(t *testing.T) {
	if got := HumanSize(512); got != "512 B" {
		t.Fatalf("got %q", got)
	}
	if got := HumanSize(2048); got != "2.0 KB" {
		t.Fatalf("got %q", got)
	}
}
