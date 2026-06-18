package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestFlagSet() *flag.FlagSet {
	fs := new(flag.FlagSet)
	fs.SetOutput(io.Discard)
	return fs
}

func newTestGitDir(t *testing.T) string {
	baseDir := t.TempDir()
	gitDir := filepath.Join(baseDir, ".git/")
	require.NoError(t, os.Mkdir(gitDir, 0700))
	return gitDir
}
