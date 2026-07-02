package main

import (
	"bytes"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaronbittel/ekko/internal/objects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashObject(t *testing.T) {
	t.Run("invalid flag", func(t *testing.T) {
		cmd := NewHashObjectCmd(newTestFlagSet())
		err := cmd.Run(io.Discard, "-t", "bla")
		assert.ErrorContains(t, err, "invalid object type \"bla\"")
	})

	t.Run("create object in db", func(t *testing.T) {
		changeIntoTestDir(t)

		assert.NoError(t, initRepo(io.Discard))
		require.NoError(t, os.WriteFile("test-content.txt", []byte("test content\n"), 0777))

		cmd := NewHashObjectCmd(newTestFlagSet())
		assert.NoError(t, cmd.Run(io.Discard, "-w", "test-content.txt"))

		path := filepath.Join(".git", "objects", "d6", "70460b4b4aece5915caf5c68d12f560a9fe3e4")
		require.FileExists(t, path)

		expected := []byte{
			0x78, 0x9c, 0x4a, 0xca, 0xc9, 0x4f, 0x52, 0x30,
			0x34, 0x66, 0x28, 0x49, 0x2d, 0x2e, 0x51, 0x48,
			0xce, 0xcf, 0x2b, 0x49, 0xcd, 0x2b, 0xe1, 0x02,
			0x04, 0x00, 0x00, 0xff, 0xff, 0x4b, 0xdf, 0x07,
			0x09,
		}

		got, err := os.ReadFile(path)
		require.NoError(t, err)

		assert.Equal(t, expected, got)
	})

	t.Run("use stdin", func(t *testing.T) {
		old := os.Stdin
		defer func() {
			os.Stdin = old
		}()

		pr, pw, err := os.Pipe()
		require.NoError(t, err)

		os.Stdin = pr

		go func() {
			_, err := pw.Write([]byte("what is up, doc?"))
			require.NoError(t, err)
			pw.Close()
		}()

		out := new(bytes.Buffer)

		cmd := NewHashObjectCmd(newTestFlagSet())
		require.NoError(t, cmd.Run(out, "--stdin"))

		assert.Equal(t, "bd9dbf5aae1a3862dd1526723246b20206e5fc37", strings.TrimSpace(out.String()))
	})
}

func TestRunHashObject(t *testing.T) {
	t.Run("hashes blob content", func(t *testing.T) {
		want := "bd9dbf5aae1a3862dd1526723246b20206e5fc37"
		obj, err := objects.BlobFromReader(strings.NewReader("what is up, doc?"))
		require.NoError(t, err)
		defer obj.Close()
		hashBytes, err := obj.Write(io.Discard)
		require.NoError(t, err)
		assert.Equal(t, want, hex.EncodeToString(hashBytes))
	})
}
