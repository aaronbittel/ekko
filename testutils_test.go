package main

import (
	"encoding/hex"
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

func newTestSha1(t *testing.T, s string) gitSha1 {
	var out gitSha1

	b, err := hex.DecodeString(s)
	require.NoError(t, err)

	if len(b) != 20 {
		t.Fatalf("invalid sha1 length: %d", len(b))
	}

	copy(out[:], b)
	return out
}
