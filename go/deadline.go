package main

import (
	"strings"
	"time"

	"homework/internal/dateutil"
	"homework/internal/model"
)

// deadlineIndicator represents the calendar interval between creation and due
// date, capped at seven circles. It does not imply percentage of work done.
func deadlineIndicator(task model.Task, now time.Time) (string, bool) {
	if task.Due == "" || task.CreatedAt.IsZero() || task.Done {
		return "", false
	}
	remaining, err := dateutil.DaysUntil(task.Due, now)
	if err != nil {
		return "", false
	}
	duration, err := dateutil.DaysUntil(task.Due, task.CreatedAt.In(now.Location()))
	if err != nil {
		return "", false
	}
	total := max(1, min(7, duration))
	filled := 1
	if duration > 0 && remaining > 1 {
		filled = min(total, max(1, (total*remaining+duration-1)/duration))
	}
	return strings.Repeat("●", filled) + strings.Repeat("○", total-filled), remaining <= 1
}
