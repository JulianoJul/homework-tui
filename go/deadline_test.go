package main

import (
	"testing"
	"time"

	"homework/internal/model"
)

func TestDeadlineIndicator(t *testing.T) {
	created := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	task := model.Task{CreatedAt: created, Due: "2026-10-27"}
	cases := []struct {
		day     int
		circles string
		urgent  bool
	}{
		{20, "●●●●●●●", false},
		{23, "●●●●○○○", false},
		{25, "●●○○○○○", false},
		{26, "●○○○○○○", true},
		{27, "●○○○○○○", true},
		{28, "●○○○○○○", true},
	}
	for _, c := range cases {
		got, urgent := deadlineIndicator(task, time.Date(2026, 10, c.day, 8, 0, 0, 0, time.UTC))
		if got != c.circles || urgent != c.urgent {
			t.Errorf("day %d: got %q, %v", c.day, got, urgent)
		}
	}
	task.Due = ""
	if got, _ := deadlineIndicator(task, created); got != "" {
		t.Fatal("task without date has indicator")
	}
	task.Due, task.Done = "2026-10-27", true
	if got, _ := deadlineIndicator(task, created); got != "" {
		t.Fatal("completed task has indicator")
	}
}

func TestDeadlineIndicatorShortAndInvalid(t *testing.T) {
	created := time.Date(2026, 10, 25, 23, 0, 0, 0, time.UTC)
	task := model.Task{CreatedAt: created, Due: "2026-10-27"}
	got, urgent := deadlineIndicator(task, created)
	if got != "●●" || urgent {
		t.Fatalf("got %q, %v", got, urgent)
	}
	task.Due = "not a date"
	if got, _ := deadlineIndicator(task, created); got != "" {
		t.Fatal("invalid date has indicator")
	}
}
