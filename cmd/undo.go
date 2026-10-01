package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newUndoCmd(store internal.Store) *cobra.Command {
	var tag string

	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Reverse the last action for a tag",
		Long: `Undo reverses the most recent start or stop for the given tag:

  Last action was 'start' (session is open)
    -> the session is deleted, as if the start never happened.
       Use this when you logged a start but then walked away.

  Last action was 'stop' (session is closed)
    -> stopped_at is cleared, reopening the session.
       Use this when you logged a stop but never actually left.`,
		Example: `  ztime undo
  ztime undo --tag lunch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := store.UndoLast(tag)
			if err == internal.ErrEntryNotFound {
				return fmt.Errorf("no %q entries to undo", tag)
			}
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			switch result.Action {
			case "deleted":
				fmt.Fprintf(out, "undid start: deleted %s entry #%d (was started at %s)\n",
					result.Entry.Tag, result.Entry.ID,
					result.Entry.StartedAt.Format(displayLayout))
			case "reopened":
				fmt.Fprintf(out, "undid stop: reopened %s entry #%d (started at %s, now open)\n",
					result.Entry.Tag, result.Entry.ID,
					result.Entry.StartedAt.Format(displayLayout))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&tag, "tag", "t", "work", "Tag of the entry to undo (default: work)")

	return cmd
}
