// Package model contains the persisted data model shared by the TUI and the
// future Python client.
package model

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	DataVersion = 1

	PriorityHigh   = "alta"
	PriorityMedium = "media"
	PriorityLow    = "baja"
)

// Subject is a university course. Folder is stored relative to the configured
// materials root (or as an absolute path when the user explicitly chooses one).
type Subject struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Folder    string    `json:"folder"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// Task is a pending piece of work for a subject. Due is an ISO date
// (YYYY-MM-DD) or empty when the task has no deadline.
type Task struct {
	ID        string     `json:"id"`
	SubjectID string     `json:"subject_id"`
	Title     string     `json:"title"`
	Due       string     `json:"due,omitempty"`
	Priority  string     `json:"priority,omitempty"`
	Notes     string     `json:"notes,omitempty"`
	Done      bool       `json:"done"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
	DoneAt    *time.Time `json:"done_at,omitempty"`
}

// Data is the complete contents of the JSON store.
type Data struct {
	Version  int       `json:"version"`
	Subjects []Subject `json:"subjects"`
	Tasks    []Task    `json:"tasks"`
}

func NewData() *Data {
	return &Data{Version: DataVersion, Subjects: []Subject{}, Tasks: []Task{}}
}

// NewID returns a short, random, URL-safe identifier.
func NewID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// Normalize repairs data that was created by an older version or edited by
// hand. It reports whether anything changed.
func (d *Data) Normalize(now time.Time) bool {
	changed := false
	if d.Version != DataVersion {
		d.Version = DataVersion
		changed = true
	}
	if d.Subjects == nil {
		d.Subjects = []Subject{}
		changed = true
	}
	if d.Tasks == nil {
		d.Tasks = []Task{}
		changed = true
	}

	ids := make(map[string]bool, len(d.Subjects))
	for i := range d.Subjects {
		s := &d.Subjects[i]
		if s.ID == "" || ids[s.ID] {
			s.ID = NewID()
			changed = true
		}
		ids[s.ID] = true
		s.Name = strings.TrimSpace(s.Name)
		s.Folder = strings.TrimSpace(s.Folder)
		if s.Name == "" && s.Folder != "" {
			s.Name = PrettyName(s.Folder)
			changed = true
		}
		if s.Folder == "" && s.Name != "" {
			s.Folder = strings.ReplaceAll(strings.ToLower(s.Name), " ", "_")
			changed = true
		}
		if s.CreatedAt.IsZero() {
			s.CreatedAt = now
			changed = true
		}
	}

	seenTasks := make(map[string]bool, len(d.Tasks))
	for i := range d.Tasks {
		t := &d.Tasks[i]
		if t.ID == "" || seenTasks[t.ID] {
			t.ID = NewID()
			changed = true
		}
		seenTasks[t.ID] = true
		t.Title = strings.TrimSpace(t.Title)
		t.Notes = strings.TrimSpace(t.Notes)
		t.Due = strings.TrimSpace(t.Due)
		normalized := NormalizePriority(t.Priority)
		if normalized != t.Priority {
			t.Priority = normalized
			changed = true
		}
		if t.CreatedAt.IsZero() {
			t.CreatedAt = now
			changed = true
		}
		if t.Done && t.DoneAt == nil {
			t.DoneAt = &now
			changed = true
		}
		if !t.Done && t.DoneAt != nil {
			t.DoneAt = nil
			changed = true
		}
	}
	return changed
}

func NormalizePriority(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "alta", "high", "h", "1":
		return PriorityHigh
	case "baja", "low", "l", "3":
		return PriorityLow
	default:
		return PriorityMedium
	}
}

func PriorityRank(p string) int {
	switch NormalizePriority(p) {
	case PriorityHigh:
		return 0
	case PriorityMedium:
		return 1
	default:
		return 2
	}
}

func PriorityLabel(p string) string {
	switch NormalizePriority(p) {
	case PriorityHigh:
		return "alta"
	case PriorityLow:
		return "baja"
	default:
		return "media"
	}
}

// PrettyName turns a folder name such as "matematica_superior_i" into
// "Matematica Superior I".
func PrettyName(folder string) string {
	folder = strings.TrimSpace(folder)
	if i := strings.LastIndexAny(folder, `/\`); i >= 0 {
		folder = folder[i+1:]
	}
	folder = strings.ReplaceAll(folder, "_", " ")
	folder = strings.ReplaceAll(folder, "-", " ")
	words := strings.Fields(folder)
	for i, w := range words {
		runes := []rune(strings.ToLower(w))
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// MergeFolders adds discovered folders that are not represented yet and
// reports how many subjects were added.
func (d *Data) MergeFolders(folders []string, now time.Time) int {
	known := make(map[string]bool, len(d.Subjects))
	for _, s := range d.Subjects {
		known[s.Folder] = true
	}
	added := 0
	for _, folder := range folders {
		if folder == "" || known[folder] {
			continue
		}
		d.Subjects = append(d.Subjects, Subject{
			ID:        NewID(),
			Name:      PrettyName(folder),
			Folder:    folder,
			CreatedAt: now,
		})
		known[folder] = true
		added++
	}
	return added
}

func (d *Data) Subject(id string) *Subject {
	for i := range d.Subjects {
		if d.Subjects[i].ID == id {
			return &d.Subjects[i]
		}
	}
	return nil
}

func (d *Data) Task(id string) *Task {
	for i := range d.Tasks {
		if d.Tasks[i].ID == id {
			return &d.Tasks[i]
		}
	}
	return nil
}

func (d *Data) SubjectIndex(id string) int {
	for i := range d.Subjects {
		if d.Subjects[i].ID == id {
			return i
		}
	}
	return -1
}

func (d *Data) TaskIndex(id string) int {
	for i := range d.Tasks {
		if d.Tasks[i].ID == id {
			return i
		}
	}
	return -1
}

// TasksFor returns a sorted copy of the tasks for one subject.
func (d *Data) TasksFor(subjectID string) []Task {
	tasks := make([]Task, 0)
	for _, t := range d.Tasks {
		if t.SubjectID == subjectID {
			tasks = append(tasks, t)
		}
	}
	SortTasks(tasks)
	return tasks
}

func SortTasks(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i], tasks[j]
		if a.Done != b.Done {
			return !a.Done
		}
		if a.Due != b.Due {
			if a.Due == "" {
				return false
			}
			if b.Due == "" {
				return true
			}
			return a.Due < b.Due
		}
		pa, pb := PriorityRank(a.Priority), PriorityRank(b.Priority)
		if pa != pb {
			return pa < pb
		}
		return strings.ToLower(a.Title) < strings.ToLower(b.Title)
	})
}

func (d *Data) PendingCount(subjectID string) int {
	n := 0
	for _, t := range d.Tasks {
		if t.SubjectID == subjectID && !t.Done {
			n++
		}
	}
	return n
}

func (d *Data) DoneCount(subjectID string) int {
	n := 0
	for _, t := range d.Tasks {
		if t.SubjectID == subjectID && t.Done {
			n++
		}
	}
	return n
}

func (d *Data) RemoveSubject(id string) {
	subjects := d.Subjects[:0]
	for _, s := range d.Subjects {
		if s.ID != id {
			subjects = append(subjects, s)
		}
	}
	d.Subjects = subjects

	tasks := d.Tasks[:0]
	for _, t := range d.Tasks {
		if t.SubjectID != id {
			tasks = append(tasks, t)
		}
	}
	d.Tasks = tasks
}

func (d *Data) RemoveTask(id string) {
	tasks := d.Tasks[:0]
	for _, t := range d.Tasks {
		if t.ID != id {
			tasks = append(tasks, t)
		}
	}
	d.Tasks = tasks
}
