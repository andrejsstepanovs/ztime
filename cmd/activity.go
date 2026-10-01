package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newTaskCmd(store internal.Store) *cobra.Command {
	cmd := withArguments(&cobra.Command{
		Use:   "task <text>",
		Short: "Log what you worked on",
		Long: `Task records free-form notes throughout the workday for later review
and standup preparation. If no work entry is active, the note is attached to
the latest preceding work entry and placed at that entry's stop time.`,
		Args: cobra.ArbitraryArgs,
	}, `  <text>  Required free-form description of what you worked on.
          Multiple words can be quoted or supplied as separate arguments.`)
	cmd.RunE = activityAddRunE(store, cmd)
	cmd.AddCommand(newTaskUndoCmd(store))
	return cmd
}

func activityAddRunE(store internal.Store, command *cobra.Command) func(*cobra.Command, []string) error {
	var at string
	command.Flags().StringVarP(&at, "at", "a", "", `Activity time, e.g. "10:45" or "2026-10-01T10:45"`)

	return func(cmd *cobra.Command, args []string) error {
		text := strings.TrimSpace(strings.Join(args, " "))
		if text == "" {
			return fmt.Errorf("task text is required")
		}
		occurredAt := time.Now()
		if at != "" {
			var err error
			occurredAt, err = parseTimeArg(at)
			if err != nil {
				return err
			}
		}

		entry, err := store.EntryAt("work", occurredAt)
		if err != nil {
			return err
		}
		if entry == nil {
			entry, err = store.LastEntryBefore("work", occurredAt)
			if err != nil {
				return err
			}
			if entry == nil {
				return fmt.Errorf("no work entry exists at or before %s; start work before logging activity", occurredAt.Format("2006-01-02 15:04"))
			}
			occurredAt = *entry.StoppedAt
			fmt.Fprintf(cmd.ErrOrStderr(),
				"warning: no active work at requested time; attached activity to work entry #%d at its stop time %s\n",
				entry.ID, occurredAt.Format("2006-01-02 15:04"))
		}

		activity, err := store.CreateActivity(entry.ID, occurredAt, text)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "logged task #%d at %s: %s\n",
			activity.ID, activity.OccurredAt.Format(displayLayout), activity.Text)
		return nil
	}
}

func newTasksCmd(store internal.Store) *cobra.Command {
	var dateStr, weekStr string
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List task notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dateStr == "" && weekStr == "" {
				dateStr = "today"
			}
			entryFilter, err := buildLogFilter(dateStr, weekStr, "")
			if err != nil {
				return err
			}
			activities, err := store.ListActivities(internal.ActivityFilter{
				Date: entryFilter.Date,
				From: entryFilter.From,
				To:   entryFilter.To,
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(activities) == 0 {
				fmt.Fprintln(out, "no tasks found")
				return nil
			}
			currentDate := ""
			for _, activity := range activities {
				date := activity.OccurredAt.Format(displayDateLayout)
				if date != currentDate {
					if currentDate != "" {
						fmt.Fprintln(out)
					}
					fmt.Fprintln(out, date)
					currentDate = date
				}
				fmt.Fprintf(out, "  #%d  %s  %s\n", activity.ID, activity.OccurredAt.Format(displayLayout), activity.Text)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&dateStr, "date", "d", "", `Filter by date: "today", "yesterday", or "2026-10-01" (default: today)`)
	cmd.Flags().StringVarP(&weekStr, "week", "w", "", `Filter by week: "this", "last", or "2026-W40"`)
	return cmd
}

func newTaskUndoCmd(store internal.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "undo",
		Short: "Delete the most recently logged task",
		RunE: func(cmd *cobra.Command, args []string) error {
			activity, err := store.UndoLastActivity()
			if err == internal.ErrActivityNotFound {
				return fmt.Errorf("no task to undo")
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "undid task #%d: %s\n", activity.ID, activity.Text)
			return nil
		},
	}
}
