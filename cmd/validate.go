package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newValidateCmd(store internal.Store) *cobra.Command {
	var dateStr, weekStr string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check entries for anomalies",
		Long: `Validate scans entries and reports problems:

  error   consecutive starts -- a start with no preceding stop for that tag
  error   more than 10h work per day
  error   more than 6h continuous work without a break
  error   insufficient breaks after 6h / 9h of work
  error   less than 11h rest between workdays
  error   work outside 06:00-23:00 CET, weekends, or public holidays
  warning more than the regular 8h daily working time
  warning open entry from a previous day -- never stopped

Working-time compliance rules apply to entries tagged "work". Structural data
checks apply to every tag.`,
		Example: `  ztime validate
  ztime validate --week this
  ztime validate --date 2026-10-01`,
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := buildLogFilter(dateStr, weekStr, "")
			if err != nil {
				return err
			}
			// Default to all time when no filter given (unlike log which defaults to today).
			entries, err := store.List(f)
			if err != nil {
				return err
			}

			issues := internal.Validate(entries)
			out := cmd.OutOrStdout()

			if len(issues) == 0 {
				fmt.Fprintln(out, "ok -- no issues found")
				return nil
			}

			for _, issue := range issues {
				fmt.Fprintln(out, issue.String())
			}
			return fmt.Errorf("%d issue(s) found", len(issues))
		},
	}

	cmd.Flags().StringVarP(&dateStr, "date", "d", "", `Scope to a date: "today", "yesterday", or "2026-10-01"`)
	cmd.Flags().StringVarP(&weekStr, "week", "w", "", `Scope to a week: "this", "last", or "2026-W40"`)

	return cmd
}
