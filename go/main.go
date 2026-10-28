package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"homework/internal/config"
	"homework/internal/dateutil"
	"homework/internal/discover"
	"homework/internal/materials"
	"homework/internal/model"
	"homework/internal/store"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	bg            = lipgloss.Color("#282828")
	fg            = lipgloss.Color("#ebdbb2")
	orange        = lipgloss.Color("#fe8019")
	gray          = lipgloss.Color("#928374")
	aqua          = lipgloss.Color("#8ec07c")
	green         = lipgloss.Color("#b8bb26")
	red           = lipgloss.Color("#fb4934")
	yellow        = lipgloss.Color("#fabd2f")
	purple        = lipgloss.Color("#d3869b")
	surface       = lipgloss.Color("#504945")
	titleStyle    = lipgloss.NewStyle().Foreground(orange).Bold(true)
	textStyle     = lipgloss.NewStyle().Foreground(fg)
	dimStyle      = lipgloss.NewStyle().Foreground(gray)
	pathStyle     = lipgloss.NewStyle().Foreground(aqua)
	errorStyle    = lipgloss.NewStyle().Foreground(red)
	selectedStyle = lipgloss.NewStyle().Foreground(orange).Background(surface).Bold(true)
)

type app struct {
	cfg                       config.Config
	store                     *store.Store
	data                      *model.Data
	width, height             int
	subject, task, file       int
	subjectOffset, itemOffset int
	fileOffset                int
	activity                  string // empty means files outside numbered activities
	activityInitialized       bool
	category                  int // own, teacher, other
	focus                     int // 0 subjects, 1 tasks, 2 files
	tab                       int // 0 tasks, 1 files
	showDone                  bool
	allTasks                  bool
	filter                    string
	filterTab                 int
	files                     []materials.Entry
	fileSummary               materials.Summary
	fileErr                   string
	mode                      string // empty, form, search, confirm, help
	inputs                    []textinput.Model
	field                     int
	editing                   string
	priority                  int
	formSubjectID             string
	formDone                  bool
	message                   string
	messageVersion            uint64
	confirmID                 string
	search                    textinput.Model
}

func main() {
	root := flag.String("root", "", "carpeta raíz de las materias")
	dataPath := flag.String("data", config.DataPath(), "archivo JSON de tareas")
	flag.Parse()
	cfg, err := config.Load(config.ConfigPath())
	if err != nil {
		if !config.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cfg = config.Default(config.DefaultRoot())
	}
	if *root != "" {
		cfg.Root = config.Expand(*root)
		config.Normalize(&cfg)
	}
	folders, err := discover.Subjects(cfg.Root, cfg.Ignore)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s := store.New(*dataPath)
	d, err := s.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	changed := d.Normalize(time.Now())
	if d.MergeFolders(folders, time.Now()) > 0 || changed {
		if err := s.Save(d); err != nil {
			fmt.Fprintln(os.Stderr, "No se pudo guardar:", err)
			os.Exit(1)
		}
	}
	m := &app{cfg: cfg, store: s, data: d, width: 100, height: 25}
	m.search = textinput.New()
	m.search.Prompt = "Buscar: "
	m.scan()
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type refreshTick struct{}

func nextRefresh() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return refreshTick{} })
}

func (m *app) Init() tea.Cmd { return nextRefresh() }
func (m *app) currentSubject() *model.Subject {
	if m.subject < 0 || m.subject >= len(m.data.Subjects) {
		return nil
	}
	return &m.data.Subjects[m.subject]
}
func (m *app) subjectDir() string {
	s := m.currentSubject()
	if s == nil {
		return ""
	}
	if filepath.IsAbs(s.Folder) {
		return s.Folder
	}
	return filepath.Join(m.cfg.Root, s.Folder)
}
func (m *app) tasks() []model.Task {
	var out []model.Task
	for _, t := range m.data.Tasks {
		if !m.allTasks && (m.currentSubject() == nil || t.SubjectID != m.currentSubject().ID) {
			continue
		}
		if t.Done && !m.showDone {
			continue
		}
		subjectName := ""
		if s := m.data.Subject(t.SubjectID); s != nil {
			subjectName = s.Name
		}
		if m.filter != "" && m.filterTab == 0 && !strings.Contains(strings.ToLower(t.Title+" "+t.Notes+" "+t.Due+" "+subjectName), strings.ToLower(m.filter)) {
			continue
		}
		out = append(out, t)
	}
	model.SortTasks(out)
	return out
}
func (m *app) visibleFiles() []materials.Entry {
	return m.filesFor(m.category)
}
func (m *app) filesFor(wantCategory int) []materials.Entry {
	return m.filesForCategory(wantCategory, true)
}
func (m *app) filesForCategory(wantCategory int, filtered bool) []materials.Entry {
	var out []materials.Entry
	for _, f := range m.files {
		parts := strings.Split(f.Rel, "/")
		activity := ""
		if len(parts) > 1 && strings.Trim(parts[0], "0123456789") == "" {
			activity = parts[0]
		}
		if activity != m.activity {
			continue
		}
		category := 2
		for _, part := range parts[:len(parts)-1] {
			if strings.EqualFold(part, "created_by_me") {
				category = 0
				break
			}
			if strings.EqualFold(part, "teacher_resources") {
				category = 1
				break
			}
		}
		if category != wantCategory {
			continue
		}
		if !filtered || m.filter == "" || m.filterTab != 1 || strings.Contains(strings.ToLower(f.Rel), strings.ToLower(m.filter)) {
			out = append(out, f)
		}
	}
	return out
}
func (m *app) categoryCount() int {
	if len(m.filesForCategory(2, false)) > 0 {
		return 3
	}
	return 2
}
func (m *app) normalizeCategory() {
	if m.category >= m.categoryCount() {
		m.category = 0
		m.file, m.fileOffset = 0, 0
	}
}
func (m *app) scan() {
	selectedPath := ""
	oldFiles := m.visibleFiles()
	if m.file >= 0 && m.file < len(oldFiles) {
		selectedPath = oldFiles[m.file].Path
	}
	m.files = nil
	m.fileSummary = materials.Summary{}
	m.fileErr = ""
	if m.currentSubject() == nil {
		return
	}
	files, truncated, err := materials.Scan(m.subjectDir())
	if err != nil {
		m.fileErr = err.Error()
		return
	}
	m.files = files
	summary, err := materials.Summarize(m.subjectDir(), files)
	if err != nil {
		m.fileErr = "No se pudo leer la estructura: " + err.Error()
		return
	}
	m.fileSummary = summary
	if !m.activityInitialized {
		m.activity = ""
		if len(summary.Activities) > 0 {
			m.activity = summary.Activities[0]
		}
		m.activityInitialized = true
	} else if m.activity != "" {
		found := false
		for _, name := range summary.Activities {
			if name == m.activity {
				found = true
				break
			}
		}
		if !found {
			m.activity = ""
			m.file, m.fileOffset = 0, 0
		}
	}
	m.normalizeCategory()
	m.file = clamp(m.file, len(m.visibleFiles()))
	if selectedPath != "" {
		for i, file := range m.visibleFiles() {
			if file.Path == selectedPath {
				m.file = i
				break
			}
		}
	}
	m.fileOffset = offsetFor(m.file, m.fileOffset, m.fileListHeight())
	if truncated {
		m.fileErr = "Listado limitado a 4000 archivos"
	}
}

// refresh re-reads the subject list and the selected subject's materials.
// Missing subjects are retained so their tasks are never silently deleted.
func (m *app) refresh(manual bool) {
	folders, err := discover.Subjects(m.cfg.Root, m.cfg.Ignore)
	if err != nil {
		m.message = "No se pudo releer: " + err.Error()
	} else {
		before := append([]model.Subject(nil), m.data.Subjects...)
		n := m.data.MergeFolders(folders, time.Now())
		if n > 0 && !m.save() {
			m.data.Subjects = before
		} else if manual {
			m.message = fmt.Sprintf("Carpetas releídas · %d materias nuevas", n)
		}
	}
	m.scan()
	if manual && m.fileErr != "" {
		m.message = "Materiales: " + m.fileErr
	}
}
func (m *app) save() bool {
	if err := m.store.Save(m.data); err != nil {
		m.message = "Error al guardar: " + err.Error()
		return false
	}
	return true
}
func clamp(n, length int) int {
	if length <= 0 || n < 0 {
		return 0
	}
	if n >= length {
		return length - 1
	}
	return n
}
func offsetFor(pos, offset, height int) int {
	if height < 1 {
		height = 1
	}
	if pos < offset {
		offset = pos
	}
	if pos >= offset+height {
		offset = pos - height + 1
	}
	if offset < 0 {
		return 0
	}
	return offset
}

func (m *app) subjectLines(index, sideW int) []string {
	s := m.data.Subjects[index]
	count := fmt.Sprintf("%d", m.data.PendingCount(s.ID))
	nameWidth := max(1, sideW-2-2-ansi.StringWidth(count)-1)
	names := strings.Split(ansi.Wrap(s.Name, nameWidth, " "), "\n")
	lines := make([]string, 0, len(names))
	for i, name := range names {
		prefix, suffix := "  ", ""
		if i == 0 {
			if index == m.subject {
				prefix = "▸ "
			}
			suffix = count
		}
		line := prefix + fit(strings.TrimSpace(name), nameWidth) + " " + suffix
		if index == m.subject {
			line = selectedStyle.Render(fit(line, sideW-2))
		} else {
			line = textStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

func (m *app) ensureSubjectVisible() {
	if len(m.data.Subjects) == 0 {
		m.subjectOffset = 0
		return
	}
	m.subject = clamp(m.subject, len(m.data.Subjects))
	m.subjectOffset = clamp(m.subjectOffset, len(m.data.Subjects))
	if m.subjectOffset > m.subject {
		m.subjectOffset = m.subject
	}
	height := max(1, m.height-8)
	sideW := min(34, max(24, m.width/3))
	for m.subjectOffset < m.subject {
		rows := 0
		for i := m.subjectOffset; i <= m.subject; i++ {
			rows += len(m.subjectLines(i, sideW))
		}
		if rows <= height {
			break
		}
		m.subjectOffset++
	}
}
func (m *app) move(delta int) {
	if m.focus == 0 {
		old := m.subject
		m.subject = clamp(m.subject+delta, len(m.data.Subjects))
		if old != m.subject {
			m.task, m.file, m.itemOffset, m.fileOffset = 0, 0, 0, 0
			m.activityInitialized = false
			m.category = 0
			m.filter = ""
			m.scan()
		}
		m.ensureSubjectVisible()
	} else if m.focus == 1 {
		m.task = clamp(m.task+delta, len(m.tasks()))
		m.itemOffset = offsetFor(m.task, m.itemOffset, m.taskListHeight())
	} else {
		m.file = clamp(m.file+delta, len(m.visibleFiles()))
		m.fileOffset = offsetFor(m.file, m.fileOffset, m.fileListHeight())
	}
}
func (m *app) taskPanelHeight() int { return max(6, (m.height-4)/3) }
func (m *app) taskListHeight() int  { return max(1, m.taskPanelHeight()-4) }
func (m *app) fileListHeight() int  { return max(1, (m.height-4)-m.taskPanelHeight()-8) }
func (m *app) changeActivity(delta int) {
	activities := append([]string{""}, m.fileSummary.Activities...)
	index := 0
	for i, activity := range activities {
		if activity == m.activity {
			index = i
			break
		}
	}
	index = (index + delta + len(activities)) % len(activities)
	m.activity = activities[index]
	m.file, m.fileOffset = 0, 0
	m.normalizeCategory()
}
func (m *app) switchTab(tab int) {
	m.tab = tab
	m.focus = tab + 1
}
func (m *app) taskForm(edit bool) tea.Cmd {
	s := m.currentSubject()
	if s == nil {
		m.message = "No hay materia seleccionada"
		return nil
	}
	m.mode = "form"
	m.message = ""
	m.field = 0
	m.editing = ""
	m.priority = 1
	m.formSubjectID = s.ID
	m.formDone = false
	m.inputs = make([]textinput.Model, 3)
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].Width = 48
		m.inputs[i].CharLimit = 300
	}
	m.inputs[0].Placeholder = "Título de la tarea"
	m.inputs[1].Placeholder = "hoy, mañana, 30/10 o 2026-10-30"
	m.inputs[2].Placeholder = "Notas (opcional)"
	if edit {
		tasks := m.tasks()
		if len(tasks) == 0 {
			m.mode = ""
			m.message = "No hay tarea seleccionada"
			return nil
		}
		t := tasks[m.task]
		m.editing = t.ID
		m.formSubjectID = t.SubjectID
		m.formDone = t.Done
		m.inputs[0].SetValue(t.Title)
		m.inputs[1].SetValue(t.Due)
		m.inputs[2].SetValue(t.Notes)
		m.priority = model.PriorityRank(t.Priority)
	}
	return m.inputs[0].Focus()
}
func (m *app) submitTask() {
	title := strings.TrimSpace(m.inputs[0].Value())
	if title == "" {
		m.message = "El título no puede estar vacío"
		return
	}
	due, err := dateutil.ParseDue(m.inputs[1].Value(), time.Now())
	if err != nil {
		m.message = err.Error()
		return
	}
	s := m.data.Subject(m.formSubjectID)
	if s == nil {
		m.message = "No hay materia seleccionada"
		return
	}
	priorities := []string{model.PriorityHigh, model.PriorityMedium, model.PriorityLow}
	before := append([]model.Task(nil), m.data.Tasks...)
	if m.editing == "" {
		t := model.Task{ID: model.NewID(), SubjectID: s.ID, Title: title, Due: due, Notes: strings.TrimSpace(m.inputs[2].Value()), Priority: priorities[m.priority], CreatedAt: time.Now(), Done: m.formDone}
		if t.Done {
			now := time.Now()
			t.DoneAt = &now
		}
		m.data.Tasks = append(m.data.Tasks, t)
	} else if t := m.data.Task(m.editing); t != nil {
		t.SubjectID = s.ID
		t.Title = title
		t.Due = due
		t.Notes = strings.TrimSpace(m.inputs[2].Value())
		t.Priority = priorities[m.priority]
		if m.formDone && !t.Done {
			now := time.Now()
			t.DoneAt = &now
		}
		if !m.formDone {
			t.DoneAt = nil
		}
		t.Done = m.formDone
	}
	if !m.save() {
		m.data.Tasks = before
		return
	}
	m.mode = ""
	m.filter = ""
	m.task = 0
	m.itemOffset = 0
	m.focus = 1
	m.tab = 0
	m.message = "Tarea guardada"
}

func (m *app) changeFormSubject(delta int) {
	n := len(m.data.Subjects)
	if n == 0 {
		return
	}
	i := m.data.SubjectIndex(m.formSubjectID)
	if i < 0 {
		i = 0
	} else {
		i = (i + delta + n) % n
	}
	m.formSubjectID = m.data.Subjects[i].ID
}
func (m *app) toggleTask() {
	tasks := m.tasks()
	if len(tasks) == 0 {
		return
	}
	t := m.data.Task(tasks[m.task].ID)
	if t == nil {
		return
	}
	before := *t
	t.Done = !t.Done
	if t.Done {
		now := time.Now()
		t.DoneAt = &now
	} else {
		t.DoneAt = nil
	}
	if !m.save() {
		*t = before
		return
	}
	m.task = clamp(m.task, len(m.tasks()))
	m.move(0)
	m.message = "Estado actualizado"
}
func (m *app) openFile() {
	files := m.visibleFiles()
	if len(files) == 0 {
		m.message = "No hay archivo seleccionado"
		return
	}
	if _, err := os.Stat(files[m.file].Path); os.IsNotExist(err) {
		m.scan()
		m.message = "El archivo ya no está en esa ruta; lista actualizada. Selecciona su nueva ubicación."
		return
	}
	if err := materials.Open(files[m.file].Path, m.cfg.Openers); err != nil {
		if os.IsNotExist(err) {
			m.scan()
			m.message = "El archivo ya no está en esa ruta; lista actualizada. Selecciona su nueva ubicación."
		} else {
			m.message = "No se pudo abrir: " + err.Error()
		}
	} else {
		m.message = "Abriendo " + files[m.file].Name
	}
}

type clearStatus struct{ version uint64 }

func (m *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if clear, ok := msg.(clearStatus); ok {
		if clear.version == m.messageVersion && m.mode != "form" {
			m.message = ""
		}
		return m, nil
	}
	previous := m.message
	result, cmd := m.update(msg)
	if m.message != previous {
		m.messageVersion++
		if m.message != "" && m.mode != "form" {
			version := m.messageVersion
			return result, tea.Batch(cmd, tea.Tick(4*time.Second, func(time.Time) tea.Msg { return clearStatus{version} }))
		}
	}
	return result, cmd
}

func (m *app) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshTick:
		m.refresh(false)
		return m, nextRefresh()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureSubjectVisible()
		m.itemOffset = offsetFor(m.task, m.itemOffset, m.taskListHeight())
		m.fileOffset = offsetFor(m.file, m.fileOffset, m.fileListHeight())
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.mode == "help" {
			if key == "esc" || key == "?" || key == "enter" {
				m.mode = ""
			}
			return m, nil
		}
		if m.mode == "confirm" {
			if key == "s" || key == "y" || key == "enter" {
				before := append([]model.Task(nil), m.data.Tasks...)
				m.data.RemoveTask(m.confirmID)
				if !m.save() {
					m.data.Tasks = before
				} else {
					m.message = "Tarea eliminada"
				}
				m.task = clamp(m.task, len(m.tasks()))
				m.move(0)
				m.mode = ""
			} else if key == "n" || key == "esc" {
				m.mode = ""
			}
			return m, nil
		}
		if m.mode == "search" {
			if key == "esc" {
				m.filter = ""
				m.task, m.file, m.itemOffset, m.fileOffset = 0, 0, 0, 0
				m.mode = ""
				return m, nil
			}
			if key == "enter" {
				m.mode = ""
				return m, nil
			}
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			m.filter = m.search.Value()
			m.task, m.file, m.itemOffset, m.fileOffset = 0, 0, 0, 0
			return m, cmd
		}
		if m.mode == "form" {
			switch key {
			case "esc":
				m.mode = ""
				m.message = ""
				return m, nil
			case "ctrl+s", "enter":
				m.submitTask()
				return m, nil
			case "tab", "shift+tab", "up", "down":
				if m.field < len(m.inputs) {
					m.inputs[m.field].Blur()
				}
				if key == "tab" || key == "down" {
					m.field = (m.field + 1) % 6
				} else {
					m.field = (m.field + 5) % 6
				}
				if m.field < len(m.inputs) {
					return m, m.inputs[m.field].Focus()
				}
				return m, nil
			}
			if m.field == 3 {
				if key == "left" {
					m.priority = (m.priority + 2) % 3
				}
				if key == "right" || key == " " {
					m.priority = (m.priority + 1) % 3
				}
				return m, nil
			}
			if m.field == 4 {
				if key == "left" {
					m.changeFormSubject(-1)
				}
				if key == "right" || key == " " {
					m.changeFormSubject(1)
				}
				return m, nil
			}
			if m.field == 5 {
				if key == "left" || key == "right" || key == " " {
					m.formDone = !m.formDone
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.inputs[m.field], cmd = m.inputs[m.field].Update(msg)
			return m, cmd
		}
		switch key {
		case "q", "ctrl+q":
			return m, tea.Quit
		case "?", "f1":
			m.mode = "help"
		case "tab":
			m.focus = (m.focus + 1) % 3
			if m.focus > 0 {
				m.tab = m.focus - 1
			}
		case "shift+tab":
			m.focus = (m.focus + 2) % 3
			if m.focus > 0 {
				m.tab = m.focus - 1
			}
		case "left":
			if m.focus == 2 {
				m.category = (m.category + m.categoryCount() - 1) % m.categoryCount()
				m.file, m.fileOffset = 0, 0
			} else {
				m.focus = 0
			}
		case "right":
			if m.focus == 2 {
				m.category = (m.category + 1) % m.categoryCount()
				m.file, m.fileOffset = 0, 0
			} else {
				m.focus = m.tab + 1
			}
		case "[", ",":
			if m.focus == 2 {
				m.changeActivity(-1)
			}
		case "]", ".":
			if m.focus == 2 {
				m.changeActivity(1)
			}
		case "up":
			m.move(-1)
		case "down":
			m.move(1)
		case "pgup":
			if m.focus == 0 {
				m.move(-max(1, m.height-8))
			} else if m.focus == 1 {
				m.move(-m.taskListHeight())
			} else {
				m.move(-m.fileListHeight())
			}
		case "pgdown":
			if m.focus == 0 {
				m.move(max(1, m.height-8))
			} else if m.focus == 1 {
				m.move(m.taskListHeight())
			} else {
				m.move(m.fileListHeight())
			}
		case "home":
			m.move(-1000000)
		case "end":
			m.move(1000000)
		case "1":
			m.switchTab(0)
		case "2":
			m.switchTab(1)
		case "n", "ctrl+n":
			return m, m.taskForm(false)
		case "e":
			if m.focus == 1 && m.tab == 0 {
				return m, m.taskForm(true)
			}
		case "enter":
			if m.focus == 0 {
				m.focus = 1
				m.tab = 0
			} else if m.focus == 1 {
				return m, m.taskForm(true)
			} else {
				m.openFile()
			}
		case " ":
			if m.focus == 1 && m.tab == 0 {
				m.toggleTask()
			}
		case "d", "delete":
			if m.focus == 1 && m.tab == 0 && len(m.tasks()) > 0 {
				m.confirmID = m.tasks()[m.task].ID
				m.mode = "confirm"
			}
		case "c":
			if m.focus == 1 {
				m.showDone = !m.showDone
				m.task = 0
				m.itemOffset = 0
			}
		case "g":
			m.allTasks = !m.allTasks
			m.task, m.itemOffset = 0, 0
			m.focus, m.tab = 1, 0
			if m.allTasks {
				m.message = "Mostrando tareas de todas las materias"
			} else {
				m.message = "Mostrando tareas de la materia seleccionada"
			}
		case "/", "ctrl+f":
			if m.focus == 0 {
				m.focus = m.tab + 1
			}
			if m.filterTab != m.focus-1 {
				m.filter = ""
			}
			m.filterTab = m.focus - 1
			m.mode = "search"
			m.search.SetValue(m.filter)
			return m, m.search.Focus()
		case "esc":
			m.filter = ""
			m.task, m.file, m.itemOffset, m.fileOffset = 0, 0, 0, 0
		case "r", "ctrl+r", "f5":
			m.refresh(true)
		case "o":
			if m.focus == 2 {
				m.openFile()
			}
		case "O":
			if m.currentSubject() != nil {
				if err := materials.Open(m.subjectDir(), m.cfg.Openers); err != nil {
					m.message = err.Error()
				} else {
					m.message = "Abriendo carpeta"
				}
			}
		}
	}
	return m, nil
}

func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = ansi.Truncate(s, width, "…")
	return s + strings.Repeat(" ", max(0, width-ansi.StringWidth(s)))
}
func panel(lines []string, width, height int, focused bool) string {
	innerW, innerH := width-2, height-2
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	if len(lines) > innerH {
		lines = lines[:innerH]
	}
	for i := range lines {
		lines[i] = fit(lines[i], innerW)
	}
	border := gray
	if focused {
		border = orange
	}
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(border).Width(innerW).Height(innerH).Render(strings.Join(lines, "\n"))
}
func (m *app) modal(title string, lines []string) string {
	w := min(70, max(20, m.width-8))
	var content []string
	content = append(content, titleStyle.Render(title), "")
	content = append(content, lines...)
	box := lipgloss.NewStyle().Foreground(fg).Background(bg).Border(lipgloss.NormalBorder()).BorderForeground(orange).Padding(1, 2).Width(w - 6).Render(strings.Join(content, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box, lipgloss.WithWhitespaceBackground(bg))
}
func (m *app) View() string {
	if m.width < 80 || m.height < 20 {
		return fmt.Sprintf("Amplía la terminal a al menos 80 × 20 (actual: %d × %d).", m.width, m.height)
	}
	if m.mode == "help" {
		return m.modal("AYUDA", []string{
			"Tab / flechas     Cambiar panel y seleccionar", "1 / 2             Tareas / Materiales", "g                 Todas las materias / materia actual", "Materiales: ←/→   Cambiar categoría", "Materiales: , / . Cambiar actividad", "Ctrl+N o n        Nueva tarea", "Enter / e         Editar tarea o abrir archivo", "Espacio           Completar o reabrir tarea", "d / Supr          Eliminar tarea (confirmación)", "c                 Mostrar / ocultar completadas", "Ctrl+F o /        Buscar; Esc limpia el filtro", "r / F5            Releer carpetas y archivos", "O                 Abrir carpeta de la materia", "Ctrl+Q o q        Salir", "", "Esc o ? para cerrar"})
	}
	if m.mode == "confirm" {
		return m.modal("ELIMINAR TAREA", []string{"¿Eliminar esta tarea? No se puede deshacer.", "", "S / Enter: confirmar     N / Esc: cancelar"})
	}
	if m.mode == "form" {
		labels := []string{"Título", "Fecha", "Notas"}
		lines := []string{}
		for i, input := range m.inputs {
			label := dimStyle.Render(fit(labels[i], 8))
			if i == m.field {
				label = titleStyle.Render(fit(labels[i], 8))
			}
			lines = append(lines, label+input.View())
		}
		p := []string{"alta", "media", "baja"}
		label := dimStyle.Render("Prioridad ")
		if m.field == 3 {
			label = titleStyle.Render("Prioridad ")
		}
		lines = append(lines, label+"◀ "+p[m.priority]+" ▶")
		label = dimStyle.Render("Materia   ")
		if m.field == 4 {
			label = titleStyle.Render("Materia   ")
		}
		name := "No disponible"
		if subject := m.data.Subject(m.formSubjectID); subject != nil {
			name = subject.Name
		}
		lines = append(lines, label+"◀ "+ansi.Truncate(name, 42, "…")+" ▶")
		label = dimStyle.Render("Estado    ")
		if m.field == 5 {
			label = titleStyle.Render("Estado    ")
		}
		state := "○ Pendiente"
		if m.formDone {
			state = "✓ Completada"
		}
		lines = append(lines, label+"◀ "+state+" ▶", "", dimStyle.Render("Tab/↑↓: campo · ←/→/Espacio: elegir"), dimStyle.Render("Ctrl+S/Enter: guardar · Esc: cancelar"))
		if m.message != "" {
			lines = append(lines, errorStyle.Render(m.message))
		}
		return m.modal("TAREA", lines)
	}
	bodyH := m.height - 4
	sideW := min(34, max(24, m.width/3))
	mainW := m.width - sideW - 1
	left := []string{titleStyle.Render(" MATERIAS"), ""}
	for i := m.subjectOffset; i < len(m.data.Subjects) && len(left) < bodyH-2; i++ {
		left = append(left, m.subjectLines(i, sideW)...)
	}
	if len(m.data.Subjects) == 0 {
		left = append(left, dimStyle.Render("  No hay materias"))
	}
	taskH := m.taskPanelHeight()
	fileH := bodyH - taskH
	taskLines := []string{titleStyle.Render(" TAREAS"), ""}
	fileLines := []string{titleStyle.Render(" MATERIALES") + dimStyle.Render("  [r/F5: releer]"), ""}
	if subject := m.currentSubject(); subject != nil {
		pending, done := 0, 0
		for _, task := range m.data.Tasks {
			if !m.allTasks && task.SubjectID != subject.ID {
				continue
			}
			if task.Done {
				done++
			} else {
				pending++
			}
		}
		scope := "Materia actual"
		if m.allTasks {
			scope = "Todas las materias"
		}
		taskLines[1] = dimStyle.Render(fmt.Sprintf(" %s · %d pendientes · %d completadas · g cambiar", scope, pending, done))
		tasks := m.tasks()
		if len(tasks) == 0 {
			taskLines = append(taskLines, dimStyle.Render("  No hay tareas. Ctrl+N para agregar."))
		}
		for i := m.itemOffset; i < len(tasks) && len(taskLines) < taskH-2; i++ {
			t := tasks[i]
			icon := "○"
			if t.Done {
				icon = "✓"
			}
			due, state := dateutil.FormatDue(t.Due, time.Now())
			if due == "" {
				due = "sin fecha"
			}
			indicator, urgent := deadlineIndicator(t, time.Now())
			if urgent {
				remaining, _ := dateutil.DaysUntil(t.Due, time.Now())
				if remaining == 0 {
					due = "! hoy"
				} else if remaining < 0 {
					due = "! " + due
				} else {
					due = "! mañana"
				}
			}
			prefix := "  "
			if i == m.task && m.focus == 1 {
				prefix = "▸ "
			}
			suffix := "  " + due + " · " + model.PriorityLabel(t.Priority)
			if indicator != "" {
				suffix += " " + indicator
			}
			available := max(1, mainW-2-ansi.StringWidth(prefix+icon+" "+suffix))
			taskTitle := t.Title
			if m.allTasks {
				name := "Materia desconocida"
				if s := m.data.Subject(t.SubjectID); s != nil {
					name = s.Name
				}
				taskTitle = "[" + ansi.Truncate(name, max(4, available/3), "…") + "] " + t.Title
			}
			line := prefix + icon + " " + fit(taskTitle, available) + suffix
			if i == m.task && m.focus == 1 {
				if urgent {
					base := prefix + icon + " " + fit(taskTitle, available)
					line = selectedStyle.Render(base) + lipgloss.NewStyle().Foreground(red).Background(surface).Bold(true).Render(suffix)
					line = fit(line, mainW-2)
				} else {
					line = selectedStyle.Render(fit(line, mainW-2))
				}
			} else if urgent {
				line = lipgloss.NewStyle().Foreground(red).Render(line)
			} else if state == dateutil.StateOverdue && !t.Done {
				line = lipgloss.NewStyle().Foreground(yellow).Render(line)
			} else if t.Done {
				line = lipgloss.NewStyle().Foreground(green).Render(line)
			} else {
				line = textStyle.Render(line)
			}
			taskLines = append(taskLines, line)
		}
		activity := m.activity
		if activity == "" {
			activity = "General"
		} else {
			activity = "Actividad " + activity
		}
		fileLines[1] = dimStyle.Render(" " + activity + "  [,/. actividad · ←/→ panel]")
		if m.fileErr != "" {
			fileLines[1] = errorStyle.Render(" " + m.fileErr)
		}
		innerW := mainW - 2
		count := m.categoryCount()
		colW := (innerW - (count - 1)) / count
		widths := make([]int, count)
		for i := range widths {
			widths[i] = colW
		}
		widths[count-1] = innerW - (count - 1) - colW*(count-1)
		subHeight := fileH - 4
		titles := []string{" MIS ARCHIVOS", " MATERIAL DE APOYO", " OTROS"}
		panels := make([]string, count)
		for category := 0; category < count; category++ {
			files := m.filesFor(category)
			lines := []string{titleStyle.Render(titles[category]), ""}
			if category == 1 && ansi.StringWidth(titles[category]) > widths[category]-2 {
				lines[0] = titleStyle.Render(" MATERIAL DE")
				lines[1] = titleStyle.Render(" APOYO")
			}
			if len(files) == 0 {
				lines = append(lines, dimStyle.Render("  Sin archivos"))
			}
			offset := 0
			if category == m.category {
				offset = m.fileOffset
			}
			for i := offset; i < len(files) && len(lines) < subHeight-2; i++ {
				prefix := "  "
				selected := m.focus == 2 && category == m.category && i == m.file
				if selected {
					prefix = "▸ "
				}
				name := files[i].Rel
				if m.activity != "" {
					name = strings.TrimPrefix(name, m.activity+"/")
				}
				if category == 0 {
					name = strings.TrimPrefix(name, "created_by_me/")
				}
				if category == 1 {
					name = strings.TrimPrefix(name, "teacher_resources/")
				}
				line := prefix + name
				if selected {
					line = selectedStyle.Render(fit(line, widths[category]-2))
				} else {
					line = pathStyle.Render(line)
				}
				lines = append(lines, line)
			}
			panels[category] = panel(lines, widths[category], subHeight, m.focus == 2 && m.category == category)
		}
		var columns []string
		for i, p := range panels {
			if i > 0 {
				columns = append(columns, " ")
			}
			columns = append(columns, p)
		}
		fileLines = append(fileLines, strings.Split(lipgloss.JoinHorizontal(lipgloss.Top, columns...), "\n")...)
	}
	right := lipgloss.JoinVertical(lipgloss.Left, panel(taskLines, mainW, taskH, m.focus == 1), panel(fileLines, mainW, fileH, m.focus == 2))
	header := titleStyle.Render(" ◆ HOMEWORK") + dimStyle.Render("  ·  ") + lipgloss.NewStyle().Foreground(purple).Render(dateutil.FormatHeader(time.Now()))
	footer := dimStyle.Render(" Tab panel · g todas · Espacio completar · c completadas · Ctrl+N tarea · ? ayuda")
	if m.filter != "" {
		footer = dimStyle.Render(" Filtro: " + m.filter + " · Esc limpia")
	}
	if m.message != "" {
		footer = textStyle.Render(" " + m.message)
	}
	if m.mode == "search" {
		footer = m.search.View() + dimStyle.Render("  Enter aplicar · Esc cancelar")
	}
	location := m.cfg.Root
	if tasks := m.tasks(); m.focus == 1 && m.task < len(tasks) {
		if s := m.data.Subject(tasks[m.task].SubjectID); s != nil {
			location = s.Name + " · " + tasks[m.task].Title
		}
	}
	if files := m.visibleFiles(); m.focus == 2 && m.file < len(files) {
		location = files[m.file].Rel
	}
	view := fit(header, m.width) + "\n" + fit(dimStyle.Render(" Materias y pendientes universitarios"), m.width) + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, panel(left, sideW, bodyH, m.focus == 0), " ", right) + "\n" + fit(footer, m.width) + "\n" + fit(dimStyle.Render(" "+location), m.width)
	return lipgloss.NewStyle().Foreground(fg).Background(bg).Render(view)
}
