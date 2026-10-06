package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newReportCmd(store internal.Store) *cobra.Command {
	var dateStr, weekStr string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Show work periods for company time tracking",
		Long: `Report lists only recorded work periods, grouped by day, for transfer
into the company time-tracking tool. Open periods are marked as not ready and
must be stopped before they can be entered as complete periods.`,
		Example: `  ztime report
  ztime report --date yesterday
  ztime report --week this`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dateStr == "" && weekStr == "" {
				dateStr = "today"
			}
			filter, err := buildLogFilter(dateStr, weekStr, "work")
			if err != nil {
				return err
			}
			entries, err := store.List(filter)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no work periods found")
				return nil
			}
			printReport(cmd.OutOrStdout(), entries)
			return nil
		},
	}
	cmd.Flags().StringVarP(&dateStr, "date", "d", "", `Filter by date: "today", "yesterday", or "2026-10-01" (default: today)`)
	cmd.Flags().StringVarP(&weekStr, "week", "w", "", `Filter by week: "this", "last", or "2026-W40"`)
	return cmd
}

func printReport(out interface{ Write([]byte) (int, error) }, entries []internal.Entry) {
	currentDate := ""
	var dailyTotal time.Duration
	hasOpenEntry := false
	printTotal := func() {
		if currentDate != "" {
			total := fmtDuration(dailyTotal)
			if hasOpenEntry {
				fmt.Fprintf(out, "  total  %s (running)\n", total)
			} else {
				fmt.Fprintf(out, "  total  %s\n", total)
			}
		}
	}

	for _, entry := range entries {
		date := entry.StartedAt.Format(displayDateLayout)
		if date != currentDate {
			if currentDate != "" {
				printTotal()
				fmt.Fprintln(out)
			}
			fmt.Fprintln(out, date)
			currentDate = date
			dailyTotal = 0
			hasOpenEntry = false
		}
		if entry.StoppedAt == nil {
			dur := entry.Duration()
			fmt.Fprintf(out, "  %s - open   %s (not ready to enter)\n",
				entry.StartedAt.Format(displayLayout),
				fmtDuration(dur))
			dailyTotal += dur
			hasOpenEntry = true
			continue
		}
		fmt.Fprintf(out, "  %s - %s  %s\n",
			entry.StartedAt.Format(displayLayout),
			entry.StoppedAt.Format(displayLayout),
			fmtDuration(entry.Duration()))
		dailyTotal += entry.Duration()
	}
	printTotal()
}
