package model

import (
	"testing"
	"time"
)

func TestPrettyName(t *testing.T) {
	cases := map[string]string{
		"matematica_superior_i":             "Matematica Superior I",
		"auditoria_de_sistemas_de_informacion": "Auditoria De Sistemas De Informacion",
		"topicos-avanzados":                 "Topicos Avanzados",
	}
	for in, want := range cases {
		if got := PrettyName(in); got != want {
			t.Errorf("PrettyName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSortTasks(t *testing.T) {
	tasks := []Task{
		{Title: "sin fecha", Due: "", Priority: PriorityHigh},
		{Title: "hoy", Due: "2026-10-27", Priority: PriorityLow},
		{Title: "hecha", Due: "2026-10-01", Done: true},
		{Title: "pronto", Due: "2026-10-28", Priority: PriorityMedium},
		{Title: "urgente", Due: "2026-10-27", Priority: PriorityHigh},
	}
	SortTasks(tasks)
	want := []string{"urgente", "hoy", "pronto", "sin fecha", "hecha"}
	for i, w := range want {
		if tasks[i].Title != w {
			t.Fatalf("position %d = %q, want %q (all: %+v)", i, tasks[i].Title, w, tasks)
		}
	}
}

func TestMergeFolders(t *testing.T) {
	d := NewData()
	now := time.Now()
	if n := d.MergeFolders([]string{"algebra", "fisica", "algebra"}, now); n != 2 {
		t.Fatalf("added = %d, want 2", n)
	}
	if n := d.MergeFolders([]string{"fisica", "quimica"}, now); n != 1 {
		t.Fatalf("second added = %d, want 1", n)
	}
	if len(d.Subjects) != 3 {
		t.Fatalf("subjects = %d, want 3", len(d.Subjects))
	}
	if d.Subjects[1].Name != "Fisica" {
		t.Fatalf("name = %q", d.Subjects[1].Name)
	}
}

func TestNormalize(t *testing.T) {
	d := &Data{
		Subjects: []Subject{{Name: "Álgebra", Folder: "algebra"}},
		Tasks:    []Task{{Title: "TP", Priority: "HIGH", Done: true}},
	}
	if !d.Normalize(time.Now()) {
		t.Fatal("expected changes")
	}
	if d.Version != DataVersion {
		t.Fatal("version not set")
	}
	if d.Subjects[0].ID == "" || d.Tasks[0].ID == "" {
		t.Fatal("ids not generated")
	}
	if d.Tasks[0].Priority != PriorityHigh {
		t.Fatalf("priority = %q", d.Tasks[0].Priority)
	}
	if d.Tasks[0].DoneAt == nil {
		t.Fatal("done_at not set")
	}
}
