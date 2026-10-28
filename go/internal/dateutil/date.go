// Package dateutil parses Spanish, human-friendly due dates and formats them
// for the task list.
package dateutil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type State int

const (
	StateNone State = iota
	StateOverdue
	StateToday
	StateSoon
	StateLater
)

var (
	weekdayNames = map[string]time.Weekday{
		"domingo": time.Sunday, "dom": time.Sunday,
		"lunes": time.Monday, "lun": time.Monday,
		"martes": time.Tuesday, "mar": time.Tuesday,
		"miercoles": time.Wednesday, "mie": time.Wednesday,
		"jueves": time.Thursday, "jue": time.Thursday,
		"viernes": time.Friday, "vie": time.Friday,
		"sabado": time.Saturday, "sab": time.Saturday,
	}
	monthNames = map[string]time.Month{
		"enero": time.January, "ene": time.January,
		"febrero": time.February, "feb": time.February,
		"marzo": time.March, "mar": time.March,
		"abril": time.April, "abr": time.April,
		"mayo": time.May, "may": time.May,
		"junio": time.June, "jun": time.June,
		"julio": time.July, "jul": time.July,
		"agosto": time.August, "ago": time.August,
		"septiembre": time.September, "setiembre": time.September, "sep": time.September, "sept": time.September,
		"octubre": time.October, "oct": time.October,
		"noviembre": time.November, "nov": time.November,
		"diciembre": time.December, "dic": time.December,
	}
	spanishWeekdays = []string{"dom", "lun", "mar", "mié", "jue", "vie", "sáb"}
	spanishMonths   = []string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}

	relativeDays = regexp.MustCompile(`^(?:\+|en\s+)(\d{1,4})\s*(?:d|dias?)?$`)
	dayMonthYear = regexp.MustCompile(`^(\d{1,2})\s*(?:de\s+)?([a-z]+)(?:\s+(?:de\s+)?(\d{4}))?$`)
)

// ParseDue converts friendly input into an ISO date. An empty input means "no
// deadline" and returns an empty string.
func ParseDue(input string, now time.Time) (string, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return "", nil
	}
	s := fold(raw)
	today := dateOnly(now)

	switch s {
	case "hoy", "today":
		return iso(today), nil
	case "manana", "tomorrow":
		return iso(today.AddDate(0, 0, 1)), nil
	case "pasado manana":
		return iso(today.AddDate(0, 0, 2)), nil
	}

	if wd, ok := weekdayNames[s]; ok {
		delta := (int(wd) - int(today.Weekday()) + 7) % 7
		return iso(today.AddDate(0, 0, delta)), nil
	}

	if m := relativeDays.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return iso(today.AddDate(0, 0, n)), nil
	}

	// ISO first, then Latin-American day/month formats. Full dates are parsed
	// before the year-less variants.
	fullLayouts := []string{"2006-01-02", "02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006", "2006/01/02"}
	for _, layout := range fullLayouts {
		if parsed, err := time.ParseInLocation(layout, s, now.Location()); err == nil {
			return iso(parsed), nil
		}
	}

	shortLayouts := []string{"02/01", "2/1", "02-01", "2-1"}
	for _, layout := range shortLayouts {
		if parsed, err := time.ParseInLocation(layout, s, now.Location()); err == nil {
			parsed = time.Date(today.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, now.Location())
			if parsed.Before(today.AddDate(0, 0, -30)) {
				parsed = parsed.AddDate(1, 0, 0)
			}
			return iso(parsed), nil
		}
	}

	if m := dayMonthYear.FindStringSubmatch(s); m != nil {
		day, _ := strconv.Atoi(m[1])
		month, ok := monthNames[m[2]]
		if ok {
			year := today.Year()
			if m[3] != "" {
				year, _ = strconv.Atoi(m[3])
			}
			parsed := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
			if parsed.Day() != day || parsed.Month() != month {
				return "", fmt.Errorf("fecha inválida: %s", raw)
			}
			if m[3] == "" && parsed.Before(today.AddDate(0, 0, -30)) {
				parsed = parsed.AddDate(1, 0, 0)
			}
			return iso(parsed), nil
		}
	}

	return "", fmt.Errorf("fecha inválida: %s (usa hoy, mañana, vie, 30/10, 2026-10-30 o +7)", raw)
}

// ParseISO validates an ISO date and returns it in canonical form.
func ParseISO(value string, now time.Time) (string, error) {
	return ParseDue(value, now)
}

func iso(t time.Time) string {
	y, m, d := t.Date()
	return fmt.Sprintf("%04d-%02d-%02d", y, int(m), d)
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func fold(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u",
		"ñ", "n",
		"à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u",
	)
	s = replacer.Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// DaysUntil returns the number of calendar days from now to due. A negative
// value means the date is in the past.
func DaysUntil(due string, now time.Time) (int, error) {
	if strings.TrimSpace(due) == "" {
		return 0, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", due, now.Location())
	if err != nil {
		return 0, err
	}
	a := dayNumber(dateOnly(now))
	b := dayNumber(parsed)
	return b - a, nil
}

func dayNumber(t time.Time) int {
	y, m, d := t.Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

// FormatDue returns a short Spanish label and how urgent it is.
func FormatDue(due string, now time.Time) (string, State) {
	if strings.TrimSpace(due) == "" {
		return "", StateNone
	}
	diff, err := DaysUntil(due, now)
	if err != nil {
		return due, StateLater
	}
	switch {
	case diff == 0:
		return "hoy", StateToday
	case diff == 1:
		return "mañana", StateSoon
	case diff > 1 && diff <= 6:
		return fmt.Sprintf("en %d d", diff), StateSoon
	case diff < 0 && diff == -1:
		return "ayer", StateOverdue
	case diff < -1 && diff >= -99:
		return fmt.Sprintf("hace %d d", -diff), StateOverdue
	}

	parsed, err := time.ParseInLocation("2006-01-02", due, now.Location())
	if err != nil {
		return due, StateLater
	}
	state := StateLater
	if diff <= 3 {
		state = StateSoon
	}
	label := fmt.Sprintf("%s %02d %s", spanishWeekdays[parsed.Weekday()], parsed.Day(), spanishMonths[parsed.Month()-1])
	if parsed.Year() != now.Year() {
		label += fmt.Sprintf(" %02d", parsed.Year()%100)
	}
	return label, state
}

// FormatLong renders a due date with the full year, useful in forms.
func FormatLong(due string, now time.Time) string {
	if strings.TrimSpace(due) == "" {
		return ""
	}
	parsed, err := time.ParseInLocation("2006-01-02", due, now.Location())
	if err != nil {
		return due
	}
	return fmt.Sprintf("%s %02d %s %d", spanishWeekdays[parsed.Weekday()], parsed.Day(), spanishMonths[parsed.Month()-1], parsed.Year())
}

// FormatHeader renders the date shown in the app header.
func FormatHeader(now time.Time) string {
	weekdays := []string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
	label := fmt.Sprintf("%s %d %s %d", weekdays[now.Weekday()], now.Day(), spanishMonths[now.Month()-1], now.Year())
	return capitalize(label)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	return string(r)
}
