package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newDeleteCmd(store internal.Store) *cobra.Command {
	return withArguments(&cobra.Command{
		Use:     "delete <id>",
		Short:   "Delete an entry",
		Args:    cobra.ExactArgs(1),
		Example: `  ztime delete 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q", args[0])
			}

			if err := store.Delete(id); err == internal.ErrEntryNotFound {
				return fmt.Errorf("entry #%d not found", id)
			} else if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "deleted entry #%d\n", id)
			return nil
		},
	}, `  <id>  Required numeric entry ID shown by 'ztime log' or 'ztime status'.`)
}
