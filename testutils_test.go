package main

import (
	"flag"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestFlagSet() *flag.FlagSet {
	fs := new(flag.FlagSet)
	fs.SetOutput(io.Discard)
	return fs
}

//lint:ignore U1000 useful helper
func newTestGitDir(t *testing.T) string {
	baseDir := t.TempDir()
	gitDir := filepath.Join(baseDir, ".git/")
	require.NoError(t, os.Mkdir(gitDir, 0755))
	return gitDir
}

func newSeededHash(t *testing.T, id int) string {
	t.Helper()

	r := rand.New(rand.NewSource(int64(id)))

	var sb strings.Builder
	for range 40 {
		sb.WriteByte(byte(r.Intn(256)))
	}

	return sb.String()
}

func changeIntoTestDir(t *testing.T) string {
	t.Helper()

	oldDir, err := os.Getwd()
	require.NoError(t, err)

	baseDir := t.TempDir()
	require.NoError(t, os.Chdir(baseDir))

	t.Cleanup(func() {
		os.Chdir(oldDir)
	})

	return baseDir
}
