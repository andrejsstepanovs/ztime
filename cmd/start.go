package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

const gapWarnThreshold = 2 * time.Hour

func newStartCmd(store internal.Store) *cobra.Command {
	var tag, at, taskText string
	var force bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new entry",
		Long: `Start creates a work entry. Use --task to also record what you are
starting to work on. This is equivalent to running 'ztime start' followed by
'ztime task', but both records are created together.`,
		Example: `  ztime start
  ztime start --tag lunch
  ztime start --at 09:00 --task "Monitor Lux async release"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			startedAt := time.Now()
			if at != "" {
				var err error
				startedAt, err = parseTimeArg(at)
				if err != nil {
					return err
				}
			}

			entries, err := loadEntries(store)
			if err != nil {
				return err
			}
			printExistingIssues(cmd.ErrOrStderr(), entries)
			if err := guardIssues(cmd.ErrOrStderr(), internal.ValidateStart(entries, startedAt, tag), force); err != nil {
				return err
			}

			// Warn if there's a suspicious gap since the last entry ended.
			last, err := store.LastStopped(tag)
			if err != nil {
				return err
			}
			if last != nil {
				gap := startedAt.Sub(*last.StoppedAt)
				if gap > gapWarnThreshold {
					fmt.Fprintf(cmd.ErrOrStderr(),
						"warning: %s gap since last %s entry ended at %s\n",
						fmtDuration(gap), tag, last.StoppedAt.Format(displayLayout))
				}
			}

			var entry *internal.Entry
			if taskText != "" {
				var activity *internal.Activity
				entry, activity, err = store.CreateWithActivity(tag, startedAt, taskText)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "started %s entry #%d at %s; logged task #%d: %s\n",
					entry.Tag, entry.ID, entry.StartedAt.Format(displayLayout), activity.ID, activity.Text)
				return nil
			}
			entry, err = store.Create(tag, startedAt, "")
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "started %s entry #%d at %s\n",
				entry.Tag, entry.ID, entry.StartedAt.Format(displayLayout))
			return nil
		},
	}

	cmd.Flags().StringVarP(&tag, "tag", "t", "work", "Entry tag")
	cmd.Flags().StringVarP(&at, "at", "a", "", `Override start time, e.g. "09:00" or "2026-10-01T09:00"`)
	cmd.Flags().StringVarP(&taskText, "task", "k", "", "Task you are starting to work on")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Bypass validation errors")

	return cmd
}
