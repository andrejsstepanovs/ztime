package cmd

import (
	"fmt"
	"time"
)

const displayLayout = "15:04"
const displayDateLayout = "2006-01-02"

// parseTimeArg parses --at flag values. Accepts "HH:MM" (assumes today) or
// full "2006-01-02T15:04" / "2006-01-02T15:04:05".
func parseTimeArg(s string) (time.Time, error) {
	layouts := []string{
		"15:04",
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
	}
	now := time.Now()
	for _, l := range layouts {
		t, err := time.ParseInLocation(l, s, time.Local)
		if err != nil {
			continue
		}
		// For HH:MM shorthand, inject today's date.
		if l == "15:04" {
			t = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognised time format %q (want HH:MM or 2006-01-02T15:04)", s)
}

// fmtDuration formats a duration as "Xh Ym".
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
