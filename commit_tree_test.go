package main

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommitObjectEncode(t *testing.T) {
	user1 := user{
		name:  "Bruce Wayne",
		email: "bruce.wayne@gothem.com",
	}
	sha1 := newSeededSha(t, 1)
	sha1Hex := hex.EncodeToString(sha1[:])

	tests := []struct {
		name         string
		commitObject commitObject
		want         []byte
	}{
		{
			name: "batman commit",
			commitObject: commitObject{
				treeHashHex: sha1Hex,
				author: signature{
					role:      roleAuthor,
					user:      user1,
					timestamp: timestamp{seconds: 15345353, offset: "+0200"},
				},
				committer: signature{
					role:      roleCommitter,
					user:      user1,
					timestamp: timestamp{seconds: 15345353, offset: "+0200"},
				},
				message: "I AM BATMAN!",
			},
			want: []byte(fmt.Sprintf("commit 181\x00tree %s\nauthor Bruce Wayne <bruce.wayne@gothem.com> 15345353 +0200\ncommitter Bruce Wayne <bruce.wayne@gothem.com> 15345353 +0200\n\nI AM BATMAN!\n", sha1Hex)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.commitObject.Encode()
			assert.Equal(t, tt.want, got)
		})
	}
}
