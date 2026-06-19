package main

import (
	"encoding/hex"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: make this less brittle, if I want to add more tests
func TestBuildIndexFile(t *testing.T) {
	entries := []*indexEntry{newTestIndexEntry(t)}

	got := buildIndexFile(entries)

	wantHeader := []byte{'D', 'I', 'R', 'C', 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x01}
	wantCTime := []byte{0x00, 0x00, 0x1E, 0x61, 0x00, 0x00, 0x1A, 0x0A}
	wantMTime := []byte{0x00, 0x00, 0x15, 0xB3, 0x00, 0x00, 0x11, 0x5C}
	wantDevice := []byte{0x00, 0x00, 0x00, 0x65}
	wantInode := []byte{0x00, 0x00, 0x00, 0xC8}
	wantFileMode := []byte{0x00, 0x00, 0x81, 0xA4}
	wantUserID := []byte{0x00, 0x00, 0x04, 0xD2}
	wantGroupID := []byte{0x00, 0x00, 0x26, 0x94}
	wantFileSize := []byte{0x00, 0x00, 0x00, 0x0A}
	wantObjectHash := []byte{
		0x1f, 0x7a, 0x7a, 0x47,
		0x2a, 0xbf, 0x3d, 0xd9,
		0x64, 0x3f, 0xd6, 0x15,
		0xf6, 0xda, 0x37, 0x9c,
		0x4a, 0xcb, 0x3e, 0x3a,
	}
	wantFlags := []byte{0x00, 0x08}
	wantFilename := []byte{'t', 'e', 's', 't', '.', 't', 'x', 't'}
	wantPadding := []byte{0x00, 0x00}
	wantIndexHash := []byte{
		0x9e, 0xc8, 0x51, 0x2e, 0x7c,
		0x4a, 0x50, 0xda, 0x4a, 0x29,
		0x24, 0x86, 0xf1, 0xa4, 0xcf,
		0x91, 0xe3, 0x84, 0x18, 0x12,
	}

	assert.Equal(t, wantHeader, got[:12])
	assert.Equal(t, wantCTime, got[12:20])
	assert.Equal(t, wantMTime, got[20:28])
	assert.Equal(t, wantDevice, got[28:32])
	assert.Equal(t, wantInode, got[32:36])
	assert.Equal(t, wantFileMode, got[36:40])
	assert.Equal(t, wantUserID, got[40:44])
	assert.Equal(t, wantGroupID, got[44:48])
	assert.Equal(t, wantFileSize, got[48:52])
	assert.Equal(t, wantObjectHash, got[52:72])
	assert.Equal(t, wantFlags, got[72:74])
	assert.Equal(t, wantFilename, got[74:82])
	assert.Equal(t, wantPadding, got[82:84])
	assert.Equal(t, wantIndexHash, got[84:])
}

func TestNewIndexEntry(t *testing.T) {
	tests := []struct {
		name string
		path string
		hash gitSha1
		stat *syscall.Stat_t
		want *indexEntry
	}{
		{
			name: "regular file",
			path: "test.txt",
			hash: Sha1(t, "1f7a7a472abf3dd9643fd615f6da379c4acb3e3a"),
			stat: &syscall.Stat_t{
				Dev:  0x00010305, // device ID
				Ino:  1000,       // inode
				Mode: 0o100644,   // regular file (S_IFREG | 0644)
				Uid:  3452,
				Gid:  98798,
				Size: 10,
				Ctim: syscall.Timespec{
					Sec:  7777,
					Nsec: 6666,
				},
				Mtim: syscall.Timespec{
					Sec:  5555,
					Nsec: 4444,
				},
			},
			want: &indexEntry{
				ctime:         gitTimestamp{seconds: 7777, nanoseconds: 6666},
				mtime:         gitTimestamp{seconds: 5555, nanoseconds: 4444},
				device:        0x00010305,
				inode:         1000,
				mode:          RegularFile,
				userID:        3452,
				groupID:       98798,
				fileSize:      10,
				gitSha:        Sha1(t, "1f7a7a472abf3dd9643fd615f6da379c4acb3e3a"),
				flags:         0x0008,
				extendedFlags: 0,
				path:          "test.txt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewIndexEntry(tt.path, tt.hash, tt.stat)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWriteIndexEntry(t *testing.T) {
	tests := []struct {
		name  string
		entry indexEntry
		want  []byte
	}{
		{
			name: "regular file",
			entry: indexEntry{
				ctime:         gitTimestamp{seconds: 0x6a350ad8, nanoseconds: 0x0ecae514},
				mtime:         gitTimestamp{seconds: 0x6a350ad8, nanoseconds: 0x0ecae514},
				device:        0x00010305,
				inode:         0x00585f56,
				mode:          RegularFile,
				userID:        0x000003e8,
				groupID:       0x000003e8,
				fileSize:      0x0000000a,
				gitSha:        Sha1(t, "1f7a7a472abf3dd9643fd615f6da379c4acb3e3a"),
				flags:         0x0008,
				extendedFlags: 0,
				path:          "test.txt",
			},
			want: []byte{
				0x6a, 0x35, 0x0a, 0xd8, 0x0e, 0xca, 0xe5, 0x14,
				0x6a, 0x35, 0x0a, 0xd8, 0x0e, 0xca, 0xe5, 0x14,
				0x00, 0x01, 0x03, 0x05, 0x00, 0x58, 0x5f, 0x56,
				0x00, 0x00, 0x81, 0xa4, 0x00, 0x00, 0x03, 0xe8,
				0x00, 0x00, 0x03, 0xe8, 0x00, 0x00, 0x00, 0x0a,
				0x1f, 0x7a, 0x7a, 0x47, 0x2a, 0xbf, 0x3d, 0xd9,
				0x64, 0x3f, 0xd6, 0x15, 0xf6, 0xda, 0x37, 0x9c,
				0x4a, 0xcb, 0x3e, 0x3a, 0x00, 0x08, 0x74, 0x65,
				0x73, 0x74, 0x2e, 0x74, 0x78, 0x74, 0x00, 0x00,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.Encode()
			assert.Equal(t, tt.want, got)
		})
	}
}

func newTestIndexEntry(t *testing.T) *indexEntry {
	return &indexEntry{
		ctime:         gitTimestamp{seconds: 7777, nanoseconds: 6666},
		mtime:         gitTimestamp{seconds: 5555, nanoseconds: 4444},
		device:        101,
		inode:         200,
		mode:          RegularFile,
		userID:        1234,
		groupID:       9876,
		fileSize:      10,
		gitSha:        Sha1(t, "1f7a7a472abf3dd9643fd615f6da379c4acb3e3a"),
		flags:         0x0008,
		extendedFlags: 0,
		path:          "test.txt",
	}
}

func Sha1(t *testing.T, s string) gitSha1 {
	var out gitSha1
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	copy(out[:], b)
	return out
}
