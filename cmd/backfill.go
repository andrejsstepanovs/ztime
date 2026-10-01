package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newBackfillCmd(store internal.Store) *cobra.Command {
	var tag, note string

	cmd := &cobra.Command{
		Use:   "backfill",
		Short: "Start a new entry from exactly where the last one stopped (no gap)",
		Long: `Backfill starts a new session with started_at set to the stopped_at of the
most recent session for the tag. Use this when you logged a stop but never
actually left -- it makes the record show no gap between sessions.

For a real break (lunch, coffee, etc.) use 'start' instead.`,
		Example: `  ztime backfill
  ztime backfill --note "never actually left"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			open, err := store.OpenEntry(tag)
			if err != nil {
				return err
			}
			if open != nil {
				return fmt.Errorf("entry already open for tag %q (id %d, started %s) -- stop it first",
					tag, open.ID, open.StartedAt.Format(displayLayout))
			}

			last, err := store.LastStopped(tag)
			if err != nil {
				return err
			}
			if last == nil {
				return fmt.Errorf("no previous %q entry found -- use 'start' instead", tag)
			}

			sess, err := store.Create(tag, *last.StoppedAt, note)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "backfilled %s entry #%d at %s (no gap from #%d)\n",
				sess.Tag, sess.ID, sess.StartedAt.Format(displayLayout), last.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&tag, "tag", "t", "work", "Entry tag")
	cmd.Flags().StringVarP(&note, "note", "n", "", "Note describing why you are backfilling")

	return cmd
}
