package dateutil

import (
	"testing"
	"time"
)

func fixedNow() time.Time {
	// Tuesday, 27 October 2026, 15:04.
	return time.Date(2026, 10, 27, 15, 4, 0, 0, time.Local)
}

func TestParseDue(t *testing.T) {
	now := fixedNow()
	cases := map[string]string{
		"":                 "",
		"hoy":              "2026-10-27",
		"mañana":           "2026-10-28",
		"pasado mañana":    "2026-10-29",
		"+7":               "2026-11-03",
		"en 3 dias":        "2026-10-30",
		"viernes":          "2026-10-30",
		"lunes":            "2026-11-02",
		"30/10":            "2026-10-30",
		"05/01":            "2027-01-05",
		"2026-12-01":       "2026-12-01",
		"01/12/2027":       "2027-12-01",
		"30 de octubre":    "2026-10-30",
		"30 octubre 2027":  "2027-10-30",
	}
	for input, want := range cases {
		got, err := ParseDue(input, now)
		if err != nil {
			t.Errorf("ParseDue(%q): %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("ParseDue(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseDueInvalid(t *testing.T) {
	now := fixedNow()
	for _, input := range []string{"31/02", "no es fecha", "2026-13-01"} {
		if _, err := ParseDue(input, now); err == nil {
			t.Errorf("ParseDue(%q) expected error", input)
		}
	}
}

func TestFormatDue(t *testing.T) {
	now := fixedNow()
	cases := []struct {
		due   string
		label string
		state State
	}{
		{"", "", StateNone},
		{"2026-10-26", "ayer", StateOverdue},
		{"2026-10-20", "hace 7 d", StateOverdue},
		{"2026-10-27", "hoy", StateToday},
		{"2026-10-28", "mañana", StateSoon},
		{"2026-10-30", "en 3 d", StateSoon},
		{"2026-11-03", "mar 03 nov", StateLater},
		{"2027-01-05", "mar 05 ene 27", StateLater},
	}
	for _, c := range cases {
		label, state := FormatDue(c.due, now)
		if label != c.label || state != c.state {
			t.Errorf("FormatDue(%q) = (%q, %v), want (%q, %v)", c.due, label, state, c.label, c.state)
		}
	}
}

func TestFormatHeader(t *testing.T) {
	if got, want := FormatHeader(fixedNow()), "Martes 27 oct 2026"; got != want {
		t.Fatalf("header = %q, want %q", got, want)
	}
}
