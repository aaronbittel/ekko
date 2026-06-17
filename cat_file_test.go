package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			input:       "blob 13\x00test content\n",
			wantContent: "test content\n",
		},
		{
			name:    "missing blob",
			input:   " 13\x00test content\n",
			wantErr: ErrBadFile,
		},
		{
			name:    "size too small",
			input:   "blob 12\x00test content\n",
			wantErr: ErrBadFile,
		},
		{
			name:    "size too big",
			input:   "blob 14\x00test content\n",
			wantErr: ErrBadFile,
		},
		{
			name:    "missing null byte",
			input:   "blob 13test content\n",
			wantErr: ErrBadFile,
		},
		{
			name:    "size not a number",
			input:   "blob nan\x00test content\n",
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
			content, err := readBlob([]byte(tt.input))
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, string(content), tt.wantContent)
		})
	}
}
