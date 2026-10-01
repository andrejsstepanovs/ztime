package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newStatusCmd(store internal.Store) *cobra.Command {
	var noTasks bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current entry, today's timeline, and tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
			out := cmd.OutOrStdout()

			entries, err := store.List(internal.LogFilter{Date: &today})
			if err != nil {
				return err
			}

			var activities []internal.Activity
			if !noTasks {
				activities, err = store.ListActivities(internal.ActivityFilter{Date: &today})
				if err != nil {
					return err
				}
			}

			if len(entries) == 0 {
				fmt.Fprintln(out, "no work entries today")
				return nil
			}

			for _, entry := range entries {
				if entry.IsOpen() {
					fmt.Fprintf(out, "  running  [#%d] %s  started %s  (%s elapsed)\n",
						entry.ID, entry.Tag, entry.StartedAt.Format(displayLayout), fmtDuration(entry.Duration()))
				}
			}

			totals := map[string]time.Duration{}
			for _, entry := range entries {
				totals[entry.Tag] += entry.Duration()
			}
			tasksByEntry := groupActivitiesByEntry(activities)

			fmt.Fprintln(out)
			fmt.Fprintf(out, "today  %s\n", today.Format(displayDateLayout))
			fmt.Fprintln(out, "---")
			for i, entry := range entries {
				if i > 0 && entries[i-1].StoppedAt != nil {
					breakStart := *entries[i-1].StoppedAt
					breakDuration := entry.StartedAt.Sub(breakStart)
					if breakDuration > 0 {
						breakReason := ""
						if entries[i-1].BreakNote != "" {
							breakReason = "  // " + entries[i-1].BreakNote
						}
						breakKind := "gap"
						if entries[i-1].BreakQualifies {
							breakKind = "break"
						}
						fmt.Fprintf(out, "       ·  %-5s  %s - %s  (%s)%s\n",
							breakKind,
							breakStart.Format(displayLayout),
							entry.StartedAt.Format(displayLayout),
							fmtDuration(breakDuration), breakReason)
					}
				}
				status := "open"
				if entry.StoppedAt != nil {
					status = entry.StoppedAt.Format(displayLayout)
				}
				note := ""
				if entry.Note != "" {
					note = "  // " + entry.Note
				}
				fmt.Fprintf(out, "  #%d  %-8s  %s - %-5s  %s%s\n",
					entry.ID, entry.Tag,
					entry.StartedAt.Format(displayLayout), status,
					fmtDuration(entry.Duration()), note)
				for _, task := range tasksByEntry[entry.ID] {
					fmt.Fprintf(out, "       └─ %s  [task #%d] %s\n",
						task.OccurredAt.Format(displayLayout), task.ID, task.Text)
				}
			}
			fmt.Fprintln(out, "---")
			for tag, duration := range totals {
				fmt.Fprintf(out, "  %-8s  %s\n", tag, fmtDuration(duration))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&noTasks, "no-tasks", false, "Hide task notes from the timeline")
	return cmd
}

func groupActivitiesByEntry(activities []internal.Activity) map[int64][]internal.Activity {
	grouped := make(map[int64][]internal.Activity)
	for _, activity := range activities {
		grouped[activity.EntryID] = append(grouped[activity.EntryID], activity)
	}
	return grouped
}
