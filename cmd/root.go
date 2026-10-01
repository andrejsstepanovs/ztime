package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

// newRoot creates a fresh root command wired to store.
func newRoot(store internal.Store) *cobra.Command {
	root := &cobra.Command{
		Use:           "ztime",
		Short:         "Work time tracker",
		Long:          "ztime - track your working time from the command line",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.AddCommand(
		newStartCmd(store),
		newStopCmd(store),
		newStatusCmd(store),
		newLogCmd(store),
		newEditCmd(store),
		newDeleteCmd(store),
		newBackfillCmd(store),
		newUndoCmd(store),
		newValidateCmd(store),
		newBalanceCmd(store),
		newReportCmd(store),
		newRulesCmd(),
		newTaskCmd(store),
		newTasksCmd(store),
	)
	configureHelp(root)
	return root
}

// NewRootForTest exposes newRoot for use in e2e tests.
func NewRootForTest(store internal.Store) *cobra.Command {
	return newRoot(store)
}

// Execute wires the store into all subcommands and runs the CLI.
func Execute(store internal.Store) {
	if err := newRoot(store).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ztime:", err)
		os.Exit(1)
	}
}
