// Package e2e tests the full CLI stack against a real SQLite database.
package e2e

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/cmd"
	"github.com/astepanovs/ztime/db"
)

// harness wires up a temp DB and returns a run helper.
func harness(t *testing.T) func(args ...string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	conn, err := db.OpenAt(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	store := db.NewStore(conn)

	return func(args ...string) (string, error) {
		root := cmd.NewRootForTest(store)
		buf := &bytes.Buffer{}
		root.SetOut(buf)
		root.SetErr(buf)
		root.SetArgs(args)
		err := root.Execute()
		return buf.String(), err
	}
}

func requireContains(t *testing.T, out, substr string) {
	t.Helper()
	require.True(t, strings.Contains(out, substr),
		"expected output to contain %q, got:\n%s", substr, out)
}

func TestE2E_StartStop(t *testing.T) {
	run := harness(t)

	out, err := run("start")
	require.NoError(t, err)
	requireContains(t, out, "started work entry #1")

	out, err = run("stop")
	require.NoError(t, err)
	requireContains(t, out, "stopped work entry #1")
}

func TestE2E_StartAlreadyOpen(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)

	_, err = run("start")
	require.Error(t, err)
	require.Contains(t, err.Error(), "working-time rules")
}

func TestE2E_StopNoOpenSession(t *testing.T) {
	run := harness(t)

	_, err := run("stop")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no open")
}

func TestE2E_Status(t *testing.T) {
	run := harness(t)

	out, err := run("status")
	require.NoError(t, err)
	requireContains(t, out, "no work entries today")

	_, err = run("start")
	require.NoError(t, err)

	out, err = run("status")
	require.NoError(t, err)
	requireContains(t, out, "running")
	requireContains(t, out, "work")
}

func TestE2E_Log(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)
	_, err = run("stop")
	require.NoError(t, err)

	out, err := run("log", "--date", "today")
	require.NoError(t, err)
	requireContains(t, out, "work")
}

func TestE2E_Edit(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)

	out, err := run("edit", "1", "--note", "edited note")
	require.NoError(t, err)
	requireContains(t, out, "updated entry #1")

	out, err = run("log", "--date", "today")
	require.NoError(t, err)
	requireContains(t, out, "edited note")
}

func TestE2E_Delete(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)
	_, err = run("stop")
	require.NoError(t, err)

	out, err := run("delete", "1")
	require.NoError(t, err)
	requireContains(t, out, "deleted entry #1")

	out, err = run("log", "--date", "today")
	require.NoError(t, err)
	requireContains(t, out, "no sessions found")
}

func TestE2E_DeleteNotFound(t *testing.T) {
	run := harness(t)

	_, err := run("delete", "999")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestE2E_LunchSession(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)

	_, err = run("start", "--tag", "lunch")
	require.NoError(t, err)

	_, err = run("stop", "--tag", "lunch")
	require.NoError(t, err)

	out, err := run("status")
	require.NoError(t, err)
	requireContains(t, out, "running") // work session still open
	requireContains(t, out, "lunch")
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestE2E_Resume(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)
	_, err = run("stop")
	require.NoError(t, err)

	out, err := run("backfill")
	require.NoError(t, err)
	requireContains(t, out, "backfilled work entry #2")
	requireContains(t, out, "no gap from #1")
}

func TestE2E_BackfillNoPrevious(t *testing.T) {
	run := harness(t)

	_, err := run("backfill")
	require.Error(t, err)
	requireContains(t, err.Error(), "no previous")
}

func TestE2E_LogDefaultsToToday(t *testing.T) {
	run := harness(t)

	// No sessions -- still should say "no sessions found" without needing --date.
	out, err := run("log")
	require.NoError(t, err)
	requireContains(t, out, "no sessions found")
}

func TestE2E_LogShowsTotals(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)
	_, err = run("stop")
	require.NoError(t, err)

	out, err := run("log")
	require.NoError(t, err)
	requireContains(t, out, "---")
	requireContains(t, out, "work")
}

func TestE2E_GapWarning(t *testing.T) {
	run := harness(t)

	// Create a session stopped 3h ago via direct store manipulation is tricky,
	// so use --at to simulate a large gap.
	_, err := run("start", "--at", "08:00")
	require.NoError(t, err)
	_, err = run("stop", "--at", "09:00")
	require.NoError(t, err)

	// Starting 4 hours later should trigger the gap warning.
	out, err := run("start", "--at", "13:00")
	require.NoError(t, err)
	requireContains(t, out, "warning:")
}

func TestE2E_UndoStart(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)

	out, err := run("undo")
	require.NoError(t, err)
	requireContains(t, out, "undid start")
	requireContains(t, out, "deleted work entry #1")

	// Session should be gone -- status shows nothing.
	out, err = run("status")
	require.NoError(t, err)
	requireContains(t, out, "no work entries today")
}

func TestE2E_UndoStop(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)
	_, err = run("stop")
	require.NoError(t, err)

	out, err := run("undo")
	require.NoError(t, err)
	requireContains(t, out, "undid stop")
	requireContains(t, out, "reopened work entry #1")

	// Session should be open again.
	out, err = run("status")
	require.NoError(t, err)
	requireContains(t, out, "running")
}

func TestE2E_UndoNoSessions(t *testing.T) {
	run := harness(t)

	_, err := run("undo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no \"work\" entries to undo")
}

func TestE2E_ValidateClean(t *testing.T) {
	run := harness(t)

	_, err := run("start", "--at", "09:00")
	require.NoError(t, err)
	_, err = run("stop", "--at", "12:00")
	require.NoError(t, err)

	out, err := run("validate")
	require.NoError(t, err)
	requireContains(t, out, "no issues found")
}

func TestE2E_ValidateTooLong(t *testing.T) {
	run := harness(t)

	_, err := run("start", "--at", "00:00", "--force")
	require.NoError(t, err)
	_, err = run("stop", "--at", "23:59", "--force")
	require.NoError(t, err)

	_, err = run("validate")
	require.Error(t, err)
	require.Contains(t, err.Error(), "issue(s)")
}

func TestE2E_ValidateEmpty(t *testing.T) {
	run := harness(t)

	out, err := run("validate")
	require.NoError(t, err)
	requireContains(t, out, "no issues found")
}

func TestE2E_StartForceBypassesConsecutiveStart(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)

	out, err := run("start", "--force")
	require.NoError(t, err)
	requireContains(t, out, "validation bypassed")
	requireContains(t, out, "started work entry #2")
}

func TestE2E_StopForceBypassesLongEntry(t *testing.T) {
	run := harness(t)

	_, err := run("start", "--at", "00:00", "--force")
	require.NoError(t, err)

	out, err := run("stop", "--at", "23:59")
	require.NoError(t, err)
	requireContains(t, out, "warning:")
}

func TestE2E_BalanceWeekdayPlusHours(t *testing.T) {
	run := harness(t)

	_, err := run("start", "--at", "2026-10-01T09:00")
	require.NoError(t, err)
	_, err = run("stop", "--at", "2026-10-01T18:00", "--force")
	require.NoError(t, err)

	out, err := run("balance", "--date", "2026-10-01")
	require.NoError(t, err)
	requireContains(t, out, "worked 9h 0m")
	requireContains(t, out, "balance +1h 0m")
}

func TestE2E_BalanceWeekendWorkHasZeroTarget(t *testing.T) {
	run := harness(t)

	_, err := run("start", "--at", "2026-10-03T09:00", "--force")
	require.NoError(t, err)
	_, err = run("stop", "--at", "2026-10-03T11:00", "--force")
	require.NoError(t, err)

	out, err := run("balance", "--date", "2026-10-03")
	require.NoError(t, err)
	requireContains(t, out, "target 0m")
	requireContains(t, out, "balance +2h 0m")
}

func TestE2E_ActivityDuringActiveWork(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)
	out, err := run("task", "Investigated", "checkout", "latency")
	require.NoError(t, err)
	requireContains(t, out, "logged task #1")

	out, err = run("tasks")
	require.NoError(t, err)
	requireContains(t, out, "Investigated checkout latency")

	out, err = run("status")
	require.NoError(t, err)
	requireContains(t, out, "[task #1] Investigated checkout latency")
}

func TestE2E_ActivityAfterStopAndUndo(t *testing.T) {
	run := harness(t)

	_, err := run("start")
	require.NoError(t, err)
	_, err = run("stop")
	require.NoError(t, err)

	out, err := run("task", "Prepared standup notes")
	require.NoError(t, err)
	requireContains(t, out, "warning: no active work")

	out, err = run("task", "undo")
	require.NoError(t, err)
	requireContains(t, out, "undid task #1")

	out, err = run("tasks")
	require.NoError(t, err)
	requireContains(t, out, "no tasks found")
}

func TestE2E_StartWithTaskCreatesLinkedTask(t *testing.T) {
	run := harness(t)

	out, err := run("start", "--task", "Monitor release")
	require.NoError(t, err)
	requireContains(t, out, "logged task #1")

	out, err = run("status")
	require.NoError(t, err)
	requireContains(t, out, "[task #1] Monitor release")
}

func TestE2E_StopReasonLabelsFollowingBreak(t *testing.T) {
	run := harness(t)

	_, err := run("start", "--at", "09:00")
	require.NoError(t, err)
	_, err = run("stop", "--at", "10:00", "--break", "--reason", "coffee")
	require.NoError(t, err)
	_, err = run("start", "--at", "10:20")
	require.NoError(t, err)

	out, err := run("status")
	require.NoError(t, err)
	requireContains(t, out, "break  10:00 - 10:20  (20m)  // coffee")
}

func TestE2E_ReportShowsWorkPeriods(t *testing.T) {
	run := harness(t)

	_, err := run("start", "--at", "09:00")
	require.NoError(t, err)
	_, err = run("stop", "--at", "12:00", "--break", "--reason", "lunch")
	require.NoError(t, err)
	_, err = run("start", "--at", "12:30")
	require.NoError(t, err)
	_, err = run("stop", "--at", "17:30")
	require.NoError(t, err)

	out, err := run("report")
	require.NoError(t, err)
	requireContains(t, out, "09:00 - 12:00  3h 0m")
	requireContains(t, out, "12:30 - 17:30  5h 0m")
	requireContains(t, out, "total  8h 0m")
}

func TestE2E_ReportRoundFlag(t *testing.T) {
	run := harness(t)

	_, err := run("start", "--at", "2026-10-01T09:07")
	require.NoError(t, err)
	_, err = run("stop", "--at", "2026-10-01T12:13", "--force")
	require.NoError(t, err)

	out, err := run("report", "--date", "2026-10-01", "--round", "10")
	require.NoError(t, err)
	requireContains(t, out, "09:00 - 12:20  3h 20m")
	requireContains(t, out, "total  3h 20m")

	// Without the flag the real times are shown.
	out, err = run("report", "--date", "2026-10-01")
	require.NoError(t, err)
	requireContains(t, out, "09:07 - 12:13  3h 6m")
}
