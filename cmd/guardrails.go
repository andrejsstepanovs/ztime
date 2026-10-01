package cmd

import (
	"fmt"
	"io"

	"github.com/astepanovs/ztime/internal"
)

func loadEntries(store internal.Store) ([]internal.Entry, error) {
	return store.List(internal.LogFilter{})
}

func printExistingIssues(w io.Writer, entries []internal.Entry) {
	for _, issue := range internal.Validate(entries) {
		fmt.Fprintf(w, "warning: existing data: %s\n", issue.String())
	}
}

func guardIssues(w io.Writer, issues []internal.Issue, force bool) error {
	var blocking []internal.Issue
	for _, issue := range issues {
		if issue.Severity == internal.SeverityError {
			blocking = append(blocking, issue)
		}
	}
	if len(blocking) == 0 {
		return nil
	}
	for _, issue := range blocking {
		fmt.Fprintf(w, "error: %s\n", issue.String())
	}
	if force {
		fmt.Fprintln(w, "warning: validation bypassed with --force")
		return nil
	}
	return fmt.Errorf("action would violate working-time rules; rerun with --force only when an approved exception applies")
}
