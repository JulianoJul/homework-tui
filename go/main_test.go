package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"homework/internal/config"
	"homework/internal/model"
	"homework/internal/store"
)

func testApp(t *testing.T) *app {
	t.Helper()
	root := t.TempDir()
	d := model.NewData()
	d.Subjects = append(d.Subjects, model.Subject{ID: "s", Name: "Materia", Folder: "materia"})
	return &app{cfg: config.Default(root), store: store.New(filepath.Join(root, "data.json")), data: d, width: 100, height: 25, focus: 1}
}

func TestSubjectNamesWrapAndScroll(t *testing.T) {
	m := testApp(t)
	m.width, m.height = 80, 20
	m.data.Subjects[0].Name = "Auditoria De Sistemas De Informacion"
	lines := m.subjectLines(0, 26)
	if len(lines) < 2 {
		t.Fatalf("expected multiple lines: %v", lines)
	}
	var name string
	for _, line := range lines {
		name += ansi.Strip(line) + " "
	}
	for _, word := range []string{"Auditoria", "Sistemas", "Informacion"} {
		if !strings.Contains(name, word) {
			t.Fatalf("missing %q in %q", word, name)
		}
	}
	if !strings.Contains(ansi.Strip(lines[0]), "0") {
		t.Fatal("pending count missing")
	}
	for i := 0; i < 12; i++ {
		m.data.Subjects = append(m.data.Subjects, model.Subject{ID: model.NewID(), Name: "Materia larga para comprobar desplazamiento"})
	}
	m.focus = 0
	m.move(12)
	if m.subjectOffset == 0 {
		t.Fatal("selection did not scroll")
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "▸ Materia") || !strings.Contains(view, "desplazamiento") {
		t.Fatal("selected subject is not fully visible")
	}
}

func TestPanelsAndMaterialLoading(t *testing.T) {
	m := testApp(t)
	m.data.Subjects[0].Name = "Una materia con nombre muy largo que debe truncarse"
	if err := os.MkdirAll(filepath.Join(m.cfg.Root, "materia", "subcarpeta"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.cfg.Root, "materia", "subcarpeta", "apuntes.pdf"), []byte("pdf"), 0600); err != nil {
		t.Fatal(err)
	}
	m.scan()
	m.category = 2
	view := m.View()
	for _, text := range []string{"TAREAS", "MATERIALES", "subcarpeta", " 0"} {
		if !strings.Contains(view, text) {
			t.Errorf("missing %q in view", text)
		}
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focus != 2 {
		t.Fatalf("focus = %d, want files", m.focus)
	}
}

func TestChangingSubjectLoadsMaterials(t *testing.T) {
	m := testApp(t)
	for _, dir := range []string{"materia", "otra"} {
		if err := os.MkdirAll(filepath.Join(m.cfg.Root, dir), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(m.cfg.Root, dir, dir+".pdf"), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m.data.Subjects = append(m.data.Subjects, model.Subject{ID: "b", Name: "Otra", Folder: "otra"})
	m.scan()
	m.focus = 0
	m.move(1)
	if len(m.files) != 1 || m.files[0].Name != "otra.pdf" {
		t.Fatalf("incorrect files: %+v", m.files)
	}
}

func TestMaterialPanelsAndActivityNavigation(t *testing.T) {
	m := testApp(t)
	root := filepath.Join(m.cfg.Root, "materia")
	for _, dir := range []string{"1/created_by_me", "1/teacher_resources", "2/created_by_me", "2/teacher_resources"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"1/created_by_me/mio.pdf", "1/teacher_resources/guia.pdf", "1/extra.pdf", "general.pdf"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m.scan()
	if m.activity != "1" {
		t.Fatalf("activity = %q", m.activity)
	}
	for category, name := range []string{"mio.pdf", "guia.pdf", "extra.pdf"} {
		files := m.filesFor(category)
		if len(files) != 1 || files[0].Name != name {
			t.Fatalf("category %d: %+v", category, files)
		}
	}
	m.focus = 2
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.category != 1 || m.visibleFiles()[0].Name != "guia.pdf" {
		t.Fatal("teacher panel not selected")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	if m.activity != "2" || len(m.visibleFiles()) != 0 {
		t.Fatal("empty activity not selected")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}})
	if m.activity != "" || len(m.filesFor(2)) != 1 {
		t.Fatal("general files not accessible")
	}
}

func TestOtherPanelOnlyForExistingFiles(t *testing.T) {
	m := testApp(t)
	root := filepath.Join(m.cfg.Root, "materia")
	if err := os.MkdirAll(filepath.Join(root, "1", "teacher_resources"), 0755); err != nil {
		t.Fatal(err)
	}
	m.scan()
	if m.categoryCount() != 2 || strings.Contains(m.View(), " OTROS") {
		t.Fatal("empty other panel should be hidden")
	}
	if !strings.Contains(m.View(), "MATERIAL DE APOYO") {
		t.Fatal("missing material de apoyo title")
	}
	m.focus = 2
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.category != 0 {
		t.Fatal("navigation should skip the hidden panel")
	}
	path := filepath.Join(root, "1", "extra.pdf")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	m.scan()
	if m.categoryCount() != 3 || !strings.Contains(m.View(), " OTROS") {
		t.Fatal("other panel should appear")
	}
	m.filter, m.filterTab = "no coincide", 1
	if m.categoryCount() != 3 {
		t.Fatal("filter must not hide an existing category")
	}
	m.category = 2
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	m.scan()
	if m.categoryCount() != 2 || m.category != 0 {
		t.Fatal("removed panel must reset selection")
	}
}

func TestRefreshAfterMoveAndKeepSelection(t *testing.T) {
	m := testApp(t)
	dir := filepath.Join(m.cfg.Root, "materia")
	if err := os.MkdirAll(filepath.Join(dir, "1"), 0755); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(dir, "z.pdf")
	if err := os.WriteFile(old, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	m.scan()
	m.activity = ""
	m.category = 2
	if err := os.WriteFile(filepath.Join(dir, "a.pdf"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	m.Update(refreshTick{})
	if len(m.files) != 2 || m.visibleFiles()[m.file].Name != "z.pdf" {
		t.Fatalf("selection was lost: %+v", m.files)
	}
	if err := os.Rename(old, filepath.Join(dir, "1", "z.pdf")); err != nil {
		t.Fatal(err)
	}
	m.openFile()
	if !strings.Contains(m.message, "lista actualizada") {
		t.Fatalf("unexpected message: %s", m.message)
	}
	if len(m.files) != 2 || m.files[0].Rel != "1/z.pdf" {
		t.Fatalf("move not detected: %+v", m.files)
	}
	if err := os.WriteFile(filepath.Join(dir, "nuevo.pdf"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyF5})
	if len(m.files) != 3 || !strings.Contains(m.message, "releídas") {
		t.Fatalf("manual reload failed: %s", m.message)
	}
}

func TestAutomaticDiscoveryKeepsMissingSubjects(t *testing.T) {
	m := testApp(t)
	if err := os.MkdirAll(filepath.Join(m.cfg.Root, "otra"), 0755); err != nil {
		t.Fatal(err)
	}
	m.Update(refreshTick{})
	if len(m.data.Subjects) != 2 {
		t.Fatalf("subjects: %+v", m.data.Subjects)
	}
	if err := os.Remove(filepath.Join(m.cfg.Root, "otra")); err != nil {
		t.Fatal(err)
	}
	m.Update(refreshTick{})
	if len(m.data.Subjects) != 2 {
		t.Fatal("missing subject should not be removed")
	}
}

func TestTaskLifecycle(t *testing.T) {
	m := testApp(t)
	m.taskForm(false)
	m.inputs[0].SetValue("Entregar informe")
	m.inputs[1].SetValue("2026-11-01")
	m.submitTask()
	if len(m.data.Tasks) != 1 || m.data.Tasks[0].Due != "2026-11-01" {
		t.Fatalf("task not saved: %+v", m.data.Tasks)
	}
	m.toggleTask()
	if !m.data.Tasks[0].Done || len(m.tasks()) != 0 {
		t.Fatal("completed task should be hidden")
	}
	m.showDone = true
	m.toggleTask()
	if m.data.Tasks[0].Done {
		t.Fatal("task should be reopened")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.data.Tasks) != 0 {
		t.Fatal("task should be deleted")
	}
	d, err := m.store.Load()
	if err != nil || len(d.Tasks) != 0 {
		t.Fatalf("persisted data: %+v, %v", d, err)
	}
}

func TestAllTasksAndSubjectSelection(t *testing.T) {
	m := testApp(t)
	m.data.Subjects = append(m.data.Subjects, model.Subject{ID: "s2", Name: "Fisica", Folder: "fisica"})
	m.taskForm(false)
	m.inputs[0].SetValue("Entrega física")
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.field != 4 {
		t.Fatalf("field = %d", m.field)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.submitTask()
	if len(m.data.Tasks) != 1 || m.data.Tasks[0].SubjectID != "s2" {
		t.Fatalf("wrong subject: %+v", m.data.Tasks)
	}
	if len(m.tasks()) != 0 {
		t.Fatal("other subject should not appear in local view")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if len(m.tasks()) != 1 || !strings.Contains(ansi.Strip(m.View()), "Fisica") {
		t.Fatal("global view should show task and subject")
	}
	m.toggleTask()
	if len(m.tasks()) != 0 || !m.data.Tasks[0].Done {
		t.Fatal("completed task should be hidden")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if len(m.tasks()) != 1 {
		t.Fatal("completed task should be visible")
	}
	m.toggleTask()
	if m.data.Tasks[0].Done {
		t.Fatal("task should reopen")
	}
	m.taskForm(true)
	if m.formSubjectID != "s2" {
		t.Fatal("editing should preserve original subject")
	}
	m.changeFormSubject(-1)
	m.submitTask()
	if m.data.Tasks[0].SubjectID != "s" {
		t.Fatal("edited subject was not saved")
	}
}

func TestFormNavigationAndCompletion(t *testing.T) {
	m := testApp(t)
	m.taskForm(false)
	m.inputs[0].SetValue("Tarea")
	for i := 0; i < 5; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.field != 5 {
		t.Fatalf("field = %d", m.field)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if !m.formDone {
		t.Fatal("status should be completed")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.data.Tasks) != 1 || !m.data.Tasks[0].Done || m.data.Tasks[0].DoneAt == nil {
		t.Fatal("completion should be saved")
	}
	m.showDone = true
	m.taskForm(true)
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.field != 5 || !m.formDone {
		t.Fatal("up should select status")
	}
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.data.Tasks[0].Done || m.data.Tasks[0].DoneAt != nil {
		t.Fatal("task should reopen")
	}
}

func TestStatusExpiryDoesNotClearNewerMessage(t *testing.T) {
	m := testApp(t)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	old := m.messageVersion
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	current := m.messageVersion
	m.Update(clearStatus{old})
	if m.message == "" {
		t.Fatal("old timer cleared newer message")
	}
	m.Update(clearStatus{current})
	if m.message != "" {
		t.Fatal("status did not expire")
	}
}

func TestInvalidTaskAndSaveFailure(t *testing.T) {
	m := testApp(t)
	m.taskForm(false)
	m.inputs[0].SetValue("Informe")
	m.inputs[1].SetValue("fecha incorrecta")
	m.submitTask()
	if len(m.data.Tasks) != 0 || m.mode != "form" {
		t.Fatal("invalid date should not save")
	}
	m.inputs[1].SetValue("")
	m.store.Path = filepath.Join(m.cfg.Root, "missing-file", "nested", "data.json")
	// A regular file cannot be used as the parent directory.
	if err := os.WriteFile(filepath.Join(m.cfg.Root, "missing-file"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	m.submitTask()
	if len(m.data.Tasks) != 0 || m.mode != "form" {
		t.Fatal("failed save should roll back")
	}
}
