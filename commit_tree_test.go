package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/aaronbittel/ekko/internal/objects"
	"github.com/stretchr/testify/assert"
)

func TestCommitObjectEncode(t *testing.T) {
	var (
		committer = objects.Signature{
			Name:  "Bruce Wayne",
			Email: "bruce.wayne@gothem.com",
			When:  time.Date(2022, time.March, 4, 0, 0, 0, 0, time.UTC),
		}
		hash = newSeededHash(t, 1)
	)

	tests := []struct {
		name   string
		commit objects.Commit
		want   []byte
	}{
		{
			name: "batman commit",
			commit: objects.Commit{
				Tree:      hash,
				Author:    committer,
				Committer: committer,
				Message:   "I AM BATMAN!",
			},
			want: fmt.Appendf(nil, "tree %s\nauthor Bruce Wayne <bruce.wayne@gothem.com> 1646352000 +0000\ncommitter Bruce Wayne <bruce.wayne@gothem.com> 1646352000 +0000\n\nI AM BATMAN!\n", hash),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.commit.Encode()
			assert.Equal(t, tt.want, got)
		})
	}
}
