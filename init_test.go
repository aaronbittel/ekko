package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitRepo(t *testing.T) {
	dir := t.TempDir()
	os.Chdir(dir)

	var sb strings.Builder

	err := initRepo(&sb)
	assert.NoError(t, err)

	wantRelGitPaths := []string{
		"HEAD",
		"branches/",
		"config",
		"description",
		"hooks/",
		"info/",
		"info/exclude",
		"objects/",
		"objects/info/",
		"objects/pack/",
		"refs/",
		"refs/heads/",
		"refs/tags/",
	}

	gitdir := filepath.Join(dir, ".git")

	for _, relpath := range wantRelGitPaths {
		p := filepath.Join(gitdir, relpath)
		if strings.HasSuffix(relpath, "/") {
			assert.DirExists(t, p)
		} else {
			assert.FileExists(t, p)
		}
	}

	data, err := os.ReadFile(filepath.Join(gitdir, "HEAD"))
	require.NoError(t, err)
	assert.Equal(t, string(data), "ref: refs/heads/main")

	gitAbspath, err := filepath.Abs(gitdir)
	require.NoError(t, err)

	want := fmt.Sprintf("Initialized empty Git repository in %s\n", gitAbspath)
	assert.Equal(t, want, sb.String())
}
