package objects

import (
	"bytes"
	"fmt"
	"time"
)

type Commit struct {
	Tree      string
	Parents   []string
	Author    Signature
	Committer Signature
	Message   string
}

type Signature struct {
	Name  string
	Email string
	When  time.Time
}

func NewCommit(tree string, parents []string, name, email, message string) *Commit {
	committer := Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}

	return &Commit{
		Tree:      tree,
		Parents:   parents,
		Author:    committer,
		Committer: committer,
		Message:   message,
	}
}

func (c Commit) Encode() []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "tree %s\n", c.Tree)

	for _, parentHash := range c.Parents {
		fmt.Fprintf(&buf, "parent %s\n", parentHash)
	}

	fmt.Fprintf(&buf, "author %s\n", c.Author)
	fmt.Fprintf(&buf, "committer %s\n", c.Committer)
	fmt.Fprintf(&buf, "\n%s\n", c.Message)

	return buf.Bytes()
}

func (s Signature) String() string {
	return fmt.Sprintf(
		"%s <%s> %d %s",
		s.Name,
		s.Email,
		s.When.Unix(),
		formatTZ(s.When),
	)
}

func formatTZ(t time.Time) string {
	_, offset := t.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}

	hours := offset / 3600
	minutes := (offset % 3600) / 60

	return fmt.Sprintf("%s%02d%02d", sign, hours, minutes)
}
