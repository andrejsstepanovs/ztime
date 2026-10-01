package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/astepanovs/ztime/internal"
)

// execute runs args against a fresh root wired to store and returns output + error.
func execute(t *testing.T, store internal.Store, args ...string) (string, error) {
	t.Helper()
	root := newRoot(store, "dev")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// requireContains asserts that out contains substr.
func requireContains(t *testing.T, out, substr string) {
	t.Helper()
	require.True(t, strings.Contains(out, substr),
		"expected output to contain %q, got:\n%s", substr, out)
}
