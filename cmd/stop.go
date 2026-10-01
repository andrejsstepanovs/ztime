package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newStopCmd(store internal.Store) *cobra.Command {
	var tag, at, reason string
	var force bool
	var legalBreak bool

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the current open entry",
		Long: `Stop closes the current open entry. --reason describes why you stopped.
Use --break only when you begin a real rest break during which you do no work.
A qualifying break must last at least 15 minutes. Commuting and gaps that are
not rest breaks must not be marked with --break.`,
		Example: `  ztime stop
	  ztime stop --break --reason "lunch"
	  ztime stop --at 17:30 --reason "commute home"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			stoppedAt := time.Now()
			if at != "" {
				var err error
				stoppedAt, err = parseTimeArg(at)
				if err != nil {
					return err
				}
			}

			entries, err := loadEntries(store)
			if err != nil {
				return err
			}
			printExistingIssues(cmd.ErrOrStderr(), entries)

			open, err := store.OpenEntry(tag)
			if err != nil {
				return err
			}
			if open == nil {
				return fmt.Errorf("no open %q entry to stop", tag)
			}

			candidateEntries := make([]internal.Entry, len(entries))
			copy(candidateEntries, entries)
			for i := range candidateEntries {
				if candidateEntries[i].ID == open.ID {
					candidateEntries[i].StoppedAt = &stoppedAt
					if reason != "" {
						candidateEntries[i].BreakNote = reason
					}
					candidateEntries[i].BreakQualifies = legalBreak
				}
			}
			// Stopping work must always remain possible. Report resulting violations,
			// but never force the user to keep working to satisfy validation.
			for _, issue := range internal.Validate(candidateEntries) {
				if issue.EntryID == open.ID && issue.Severity == internal.SeverityError {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", issue.String())
				}
			}

			entry, err := store.Stop(tag, stoppedAt, reason, legalBreak)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "stopped %s entry #%d at %s (duration: %s)\n",
				entry.Tag, entry.ID, stoppedAt.Format(displayLayout), fmtDuration(entry.Duration()))
			return nil
		},
	}

	cmd.Flags().StringVarP(&tag, "tag", "t", "work", "Tag of the open entry to stop (default: work)")
	cmd.Flags().StringVarP(&at, "at", "a", "", `Override stop time, e.g. "17:30"`)
	cmd.Flags().StringVarP(&reason, "reason", "r", "", "Reason for the break beginning at this stop")
	cmd.Flags().BoolVarP(&legalBreak, "break", "b", false, "Mark the following gap as a legally qualifying rest break")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Bypass validation errors")

	return cmd
}
