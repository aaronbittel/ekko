package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestGetObjectPath(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		wantFile string
		wantErr  error
	}{
		{
			name:     "find exact",
			files:    []string{"AA"},
			wantFile: "AA",
		},
		{
			name:     "find not exact",
			files:    []string{"AABBCCDD"},
			wantFile: "AABBCCDD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			gitRepo := filepath.Join(baseDir, ".git")
			objectsDir := filepath.Join(gitRepo, "objects", "d6")

			require.NoError(t, os.MkdirAll(objectsDir, 0o700))

			for _, file := range tt.files {
				require.NoError(t,
					os.WriteFile(filepath.Join(objectsDir, file), nil, 0o644),
				)
			}

			got, err := getObjectPath(gitRepo, "d6AA")

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t,
				filepath.Join(objectsDir, tt.wantFile),
				got,
			)
		})
	}
}

func TestMinimalUniqueObjectPrefix(t *testing.T) {
	tests := []struct {
		name   string
		inputs []string
	}{
		{
			name:   "binary collision",
			inputs: []string{"AA", "AB"},
		},
		{
			name:   "three way collision",
			inputs: []string{"AAA", "AAB", "ABC"},
		},
		{
			name:   "mixed lengths",
			inputs: []string{"AAAAAAAAAA", "AAAAAAAAAB", "AAAB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// minimalUniqueObjectPrefix expects each name to be exactly 38 chars
			inputs := make([]string, len(tt.inputs))
			for i, input := range tt.inputs {
				inputs[i] = input + strings.Repeat("X", 38-len(input))
			}

			got := minimalUniqueObjectPrefix(inputs)

			// invariant: each prefix must uniquely identify its object
			for i, prefix := range got {
				for j, full := range inputs {
					if i == j {
						continue
					}
					assert.Falsef(
						t,
						strings.HasPrefix(full, prefix),
						"prefix %q should not match full object %q",
						prefix,
						full,
					)
				}
			}
		})
	}
}
