package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadBlob(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     error
		wantContent string
	}{
		{
			name:        "valid",
			input:       "13\x00test content\n",
			wantContent: "test content\n",
		},
		{
			name:    "size too small",
			input:   "12\x00test content\n",
			wantErr: ErrBadFile,
		},
		{
			name:    "size too big",
			input:   "14\x00test content\n",
			wantErr: ErrBadFile,
		},
		{
			name:    "missing null byte",
			input:   "13test content\n",
			wantErr: ErrBadFile,
		},
		{
			name:    "size not a number",
			input:   "nan\x00test content\n",
			wantErr: ErrBadFile,
		},
		{
			name:    "empty file",
			input:   "",
			wantErr: ErrBadFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := readBlob(tt.input)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, string(content), tt.wantContent)
		})
	}
}

func TestGetObjectType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantTyp  string
		wantRest string
	}{
		{
			name:     "valid blob",
			input:    "blob rest",
			wantTyp:  typeBlob,
			wantRest: "rest",
		},
		{
			name:     "valid commit",
			input:    "commit rest",
			wantTyp:  typeCommit,
			wantRest: "rest",
		},
		{
			name:     "valid tree",
			input:    "tree rest",
			wantTyp:  typeTree,
			wantRest: "rest",
		},
		{
			name:     "valid tag",
			input:    "tag rest",
			wantTyp:  typeTag,
			wantRest: "rest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, rest, err := getObjectType(tt.input)
			assert.Nil(t, err)
			assert.Equal(t, tt.wantTyp, typ)
			assert.Equal(t, tt.wantRest, rest)
		})
	}

	t.Run("invalid type", func(t *testing.T) {
		_, _, err := getObjectType("unknown rest")
		assert.ErrorContains(t, err, "unknown object info")
	})

	t.Run("invalid format", func(t *testing.T) {
		_, _, err := getObjectType("missingspace")
		assert.ErrorIs(t, err, ErrBadFile)
	})
}

func TestLoadGitObject(t *testing.T) {
	gitDir := newTestGitDir(t)

	objectPath := filepath.Join(gitDir, ".git", "objects", "d6")
	require.NoError(t, os.MkdirAll(objectPath, 0700))
	objectName := "d670460b4b4aece5915caf5c68d12f560a9fe3e4"

	objectFile := filepath.Join(objectPath, objectName[2:])
	// compressed 'test content\n' blob
	data := []byte{
		0x78, 0x01, 0x4b, 0xca, 0xc9, 0x4f, 0x52, 0x30,
		0x34, 0x66, 0x28, 0x49, 0x2d, 0x2e, 0x51, 0x48,
		0xce, 0xcf, 0x2b, 0x49, 0xcd, 0x2b, 0xe1, 0x02,
		0x00, 0x4b, 0xdf, 0x07, 0x09,
	}
	require.NoError(t, os.WriteFile(objectFile, data, 0666))

	objectType, content, err := loadGitObject(gitDir, objectName)
	assert.Equal(t, typeBlob, objectType)
	assert.Equal(t, "13\x00test content\n", content)
	assert.Nil(t, err)
}
