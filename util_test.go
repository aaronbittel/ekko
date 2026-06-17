package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGitRepo(t *testing.T) {
	t.Run("find parent dir", func(t *testing.T) {
		baseDir := t.TempDir()
		gitDir := filepath.Join(baseDir, ".git/")
		require.NoError(t, os.Mkdir(gitDir, 0700))

		p := filepath.Join(baseDir, "a", "b", "c")
		require.NoError(t, os.MkdirAll(p, 0700))
		require.NoError(t, os.Chdir(p))

		got, err := findGitRepo()
		require.NoError(t, err)

		want, err := filepath.Abs(gitDir)
		require.NoError(t, err)

		assert.Equal(t, want, got)
	})

	t.Run("no git repo", func(t *testing.T) {
		baseDir := t.TempDir()

		p := filepath.Join(baseDir, "a", "b", "c")
		require.NoError(t, os.MkdirAll(p, 0700))
		require.NoError(t, os.Chdir(p))

		_, err := findGitRepo()
		assert.ErrorIs(t, err, ErrGitRepoNotFound)
	})
}
