package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

const displayLayout = "15:04"
const displayDateLayout = "2006-01-02"

// parseTimeArg parses --at flag values. Accepts "HH:MM" (assumes today) or
// full "2006-01-02T15:04" / "2006-01-02T15:04:05".
func parseTimeArg(s string) (time.Time, error) {
	return parseTimeArgOn(s, time.Now())
}

// parseTimeArgOn parses a time argument, anchoring a bare "HH:MM" to the
// calendar day of ref. Full timestamps are used as-is.
func parseTimeArgOn(s string, ref time.Time) (time.Time, error) {
	layouts := []string{
		"15:04",
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
	}
	for _, l := range layouts {
		t, err := time.ParseInLocation(l, s, time.Local)
		if err != nil {
			continue
		}
		// For HH:MM shorthand, inherit ref's date.
		if l == "15:04" {
			t = time.Date(ref.Year(), ref.Month(), ref.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
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

// addRoundFlag registers the shared --round display-rounding flag.
func addRoundFlag(cmd *cobra.Command, p *int) {
	cmd.Flags().IntVarP(p, "round", "r", 0,
		"Round times for display: start down, stop up, to a multiple of N minutes (e.g. 5, 10, 15, 30)")
}

// applyRounding rounds entry times for display. zero minutes disables it.
func applyRounding(entries []internal.Entry, minutes int) ([]internal.Entry, error) {
	if minutes < 0 {
		return nil, fmt.Errorf("--round must be a positive number of minutes")
	}
	if minutes == 0 {
		return entries, nil
	}
	step := time.Duration(minutes) * time.Minute
	for i := range entries {
		entries[i] = entries[i].Rounded(step)
	}
	return entries, nil
}
