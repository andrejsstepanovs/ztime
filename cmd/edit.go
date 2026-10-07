package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newEditCmd(store internal.Store) *cobra.Command {
	var tag, startedAt, stoppedAt, note, breakNote string
	var breakQualifies bool

	cmd := withArguments(&cobra.Command{
		Use:   "edit <id>",
		Short: "Edit an entry",
		Args:  cobra.ExactArgs(1),
		Example: `  ztime edit 42 -s 09:05
  ztime edit 42 -n "revised note"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q", args[0])
			}

			entry, err := store.Get(id)
			if err == internal.ErrEntryNotFound {
				return fmt.Errorf("entry #%d not found", id)
			}
			if err != nil {
				return err
			}

			p := internal.EditParams{}
			if cmd.Flags().Changed("tag") {
				p.Tag = &tag
			}
			if startedAt != "" {
				t, err := parseTimeArgOn(startedAt, entry.StartedAt)
				if err != nil {
					return fmt.Errorf("--start: %w", err)
				}
				p.StartedAt = &t
			}
			if stoppedAt != "" {
				// Anchor a bare HH:MM to the day the stop lands on: the new
				// start's day when both are edited, otherwise the existing
				// stop's day (falling back to the start's day when open).
				day := entry.StartedAt
				if p.StartedAt != nil {
					day = *p.StartedAt
				} else if entry.StoppedAt != nil {
					day = *entry.StoppedAt
				}
				t, err := parseTimeArgOn(stoppedAt, day)
				if err != nil {
					return fmt.Errorf("--stop: %w", err)
				}
				p.StoppedAt = &t
			}
			if cmd.Flags().Changed("note") {
				p.Note = &note
			}
			if cmd.Flags().Changed("break-note") {
				p.BreakNote = &breakNote
			}
			if cmd.Flags().Changed("break") {
				p.BreakQualifies = &breakQualifies
			}

			sess, err := store.Edit(id, p)
			if err == internal.ErrEntryNotFound {
				return fmt.Errorf("entry #%d not found", id)
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "updated entry #%d\n", sess.ID)
			return nil
		},
	}, `  <id>  Required numeric entry ID shown by 'ztime log' or 'ztime status'.`)

	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Change tag")
	cmd.Flags().StringVarP(&startedAt, "start", "s", "", "Change start time")
	cmd.Flags().StringVarP(&stoppedAt, "stop", "S", "", "Change stop time")
	cmd.Flags().StringVarP(&note, "note", "n", "", "Change note")
	cmd.Flags().StringVarP(&breakNote, "break-note", "b", "", "Change the reason for the following break")
	cmd.Flags().BoolVar(&breakQualifies, "break", false, "Mark the following gap as a legally qualifying rest break")

	return cmd
}
