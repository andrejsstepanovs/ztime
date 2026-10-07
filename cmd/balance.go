package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/astepanovs/ztime/internal"
)

func newBalanceCmd(store internal.Store) *cobra.Command {
	var dateStr, weekStr string
	var round int

	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Show cumulative plus or minus working time",
		Long: `Balance compares recorded work with an 8-hour target on each logged
ordinary weekday. Days without work entries are ignored. Logged weekend and
nationwide German public-holiday work has a zero target and therefore counts
entirely as positive balance.`,
		Example: `  ztime balance
  ztime balance --week this
  ztime balance --date today
  ztime balance --round 30`,
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, err := buildLogFilter(dateStr, weekStr, "")
			if err != nil {
				return err
			}
			entries, err := store.List(filter)
			if err != nil {
				return err
			}
			entries, err = applyRounding(entries, round)
			if err != nil {
				return err
			}

			report := internal.CalculateBalance(entries)
			out := cmd.OutOrStdout()
			if len(report.Days) == 0 {
				fmt.Fprintln(out, "no work entries found")
				return nil
			}

			for _, day := range report.Days {
				fmt.Fprintf(out, "%s  worked %s  target %s  balance %s\n",
					day.Date.Format(displayDateLayout),
					fmtDuration(day.Worked),
					fmtDuration(day.Target),
					fmtSignedDuration(day.Balance))
			}
			fmt.Fprintln(out, "---")
			fmt.Fprintf(out, "worked %s  target %s  balance %s\n",
				fmtDuration(report.Worked),
				fmtDuration(report.Target),
				fmtSignedDuration(report.Balance))
			return nil
		},
	}

	cmd.Flags().StringVarP(&dateStr, "date", "d", "", `Scope to a date: "today", "yesterday", or "2026-10-01"`)
	cmd.Flags().StringVarP(&weekStr, "week", "w", "", `Scope to a week: "this", "last", or "2026-W40"`)
	addRoundFlag(cmd, &round)
	return cmd
}

func fmtSignedDuration(duration time.Duration) string {
	sign := "+"
	if duration < 0 {
		sign = "-"
		duration = -duration
	}
	return sign + fmtDuration(duration)
}
