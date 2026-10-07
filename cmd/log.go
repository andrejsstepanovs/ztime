package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newLogCmd(store internal.Store) *cobra.Command {
	var dateStr, weekStr, tag string
	var asJSON bool
	var round int

	cmd := &cobra.Command{
		Use:   "log",
		Short: "List past sessions",
		Example: `  ztime log
  ztime log --date yesterday
  ztime log --week last
  ztime log --tag work --json
  ztime log --round 15`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to today when no date/week filter is given.
			if dateStr == "" && weekStr == "" {
				dateStr = "today"
			}

			f, err := buildLogFilter(dateStr, weekStr, tag)
			if err != nil {
				return err
			}

			sessions, err := store.List(f)
			if err != nil {
				return err
			}
			sessions, err = applyRounding(sessions, round)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()

			if asJSON {
				return printJSON(out, sessions)
			}

			if len(sessions) == 0 {
				fmt.Fprintln(out, "no sessions found")
				return nil
			}

			printSessions(out, sessions)
			return nil
		},
	}

	cmd.Flags().StringVarP(&dateStr, "date", "d", "", `Filter by date: "today", "yesterday", or "2026-10-01" (default: today)`)
	cmd.Flags().StringVarP(&weekStr, "week", "w", "", `Filter by week: "this", "last", or "2026-W40"`)
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Filter by tag")
	cmd.Flags().BoolVarP(&asJSON, "json", "j", false, "Output as JSON")
	addRoundFlag(cmd, &round)

	return cmd
}

func buildLogFilter(dateStr, weekStr, tag string) (internal.LogFilter, error) {
	f := internal.LogFilter{Tag: tag}

	if dateStr != "" && weekStr != "" {
		return f, fmt.Errorf("--date and --week are mutually exclusive")
	}

	if dateStr != "" {
		t, err := parseDate(dateStr)
		if err != nil {
			return f, err
		}
		f.Date = &t
	}

	if weekStr != "" {
		from, to, err := parseWeek(weekStr)
		if err != nil {
			return f, err
		}
		f.From = &from
		f.To = &to
	}

	return f, nil
}

func parseDate(s string) (time.Time, error) {
	now := time.Now()
	switch strings.ToLower(s) {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local), nil
	case "yesterday":
		y := now.AddDate(0, 0, -1)
		return time.Date(y.Year(), y.Month(), y.Day(), 0, 0, 0, 0, time.Local), nil
	default:
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			return time.Time{}, fmt.Errorf("unrecognised date %q (want today/yesterday/YYYY-MM-DD)", s)
		}
		return t, nil
	}
}

func parseWeek(s string) (from, to time.Time, err error) {
	now := time.Now()
	switch strings.ToLower(s) {
	case "this":
		from, to = weekBounds(now)
	case "last":
		from, to = weekBounds(now.AddDate(0, 0, -7))
	default:
		var year, week int
		if _, err = fmt.Sscanf(s, "%d-W%d", &year, &week); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("unrecognised week %q (want this/last/YYYY-Www)", s)
		}
		jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.Local)
		_, w := jan4.ISOWeek()
		mon := jan4.AddDate(0, 0, (week-w)*7-int(jan4.Weekday())+1)
		from, to = weekBounds(mon)
	}
	return
}

func weekBounds(t time.Time) (from, to time.Time) {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	mon := t.AddDate(0, 0, -(wd - 1))
	from = time.Date(mon.Year(), mon.Month(), mon.Day(), 0, 0, 0, 0, time.Local)
	to = from.AddDate(0, 0, 6)
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.Local)
	return
}

// printSessions renders sessions grouped by day, with per-tag and grand totals per day.
func printSessions(out io.Writer, sessions []internal.Entry) {
	// Group sessions by calendar day.
	type dayGroup struct {
		date     string
		sessions []internal.Entry
	}
	var groups []dayGroup
	var cur *dayGroup

	for _, s := range sessions {
		day := s.StartedAt.Format(displayDateLayout)
		if cur == nil || cur.date != day {
			groups = append(groups, dayGroup{date: day})
			cur = &groups[len(groups)-1]
		}
		cur.sessions = append(cur.sessions, s)
	}

	for i, g := range groups {
		if i > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, g.date)

		tagTotals := map[string]time.Duration{}
		for _, s := range g.sessions {
			stopped := "open"
			if s.StoppedAt != nil {
				stopped = s.StoppedAt.Format(displayLayout)
			}
			note := ""
			if s.Note != "" {
				note = "  // " + s.Note
			}
			fmt.Fprintf(out, "  #%-4d  %-8s  %s - %-5s  %s%s\n",
				s.ID, s.Tag,
				s.StartedAt.Format(displayLayout), stopped,
				fmtDuration(s.Duration()), note)
			tagTotals[s.Tag] += s.Duration()
		}

		// Totals footer.
		var grand time.Duration
		for _, d := range tagTotals {
			grand += d
		}
		fmt.Fprintln(out, "  ---")
		for tag, d := range tagTotals {
			fmt.Fprintf(out, "  %-8s  %s\n", tag, fmtDuration(d))
		}
		if len(tagTotals) > 1 {
			fmt.Fprintf(out, "  %-8s  %s\n", "total", fmtDuration(grand))
		}
	}
}

func printJSON(out io.Writer, sessions []internal.Entry) error {
	type row struct {
		ID        int64   `json:"id"`
		Tag       string  `json:"tag"`
		StartedAt string  `json:"started_at"`
		StoppedAt *string `json:"stopped_at"`
		Note      string  `json:"note,omitempty"`
		Duration  string  `json:"duration"`
	}
	var rows []row
	for _, s := range sessions {
		r := row{
			ID:        s.ID,
			Tag:       s.Tag,
			StartedAt: s.StartedAt.Format(time.RFC3339),
			Note:      s.Note,
			Duration:  fmtDuration(s.Duration()),
		}
		if s.StoppedAt != nil {
			v := s.StoppedAt.Format(time.RFC3339)
			r.StoppedAt = &v
		}
		rows = append(rows, r)
	}
	// Fall back to real stdout when out is discarded (shouldn't happen in practice).
	w := out
	if w == nil {
		w = os.Stdout
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}
