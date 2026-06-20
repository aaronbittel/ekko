package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type WriteTreeCmd struct {
	description string

	fs *flag.FlagSet
}

func NewWriteTreeCmd(fs *flag.FlagSet) *WriteTreeCmd {
	cmd := &WriteTreeCmd{
		description: "Create a tree object",
		fs:          fs,
	}
	cmd.defineFlags()
	return cmd
}

func (cmd *WriteTreeCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-write-tree - Create a tree object from the current index")
		cmd.fs.PrintDefaults()
	}
}

func (cmd *WriteTreeCmd) Description() string {
	return cmd.description
}

func (cmd *WriteTreeCmd) Name() string {
	return cmd.fs.Name()
}

type treeObject struct {
	name  string
	blobs []*blobEntry
	trees []*treeObject
}

func newTreeObject(name string) *treeObject {
	return &treeObject{name: name}
}

func (t *treeObject) addBlob(mode gitMode, gitSha gitSha1, path string) {
	var blob blobEntry
	switch mode {
	case RegularFile:
		blob.mode = blobModeRegular
	case ExecutableFile:
		blob.mode = blobModeExecutable
	default:
		panic(fmt.Sprintf("illegal blob mode %v", mode))
	}

	blob.hash = gitSha
	blob.name = path

	t.blobs = append(t.blobs, &blob)
}

func (t *treeObject) addTree(name string) *treeObject {
	next := newTreeObject(name)
	t.trees = append(t.trees, next)
	return next
}

type blobMode []byte

var (
	blobModeRegular    = []byte("100644")
	blobModeExecutable = []byte("100755")
)

type blobEntry struct {
	mode blobMode
	hash gitSha1
	name string
}

func (b blobEntry) Encode() []byte {
	buf := make([]byte, 0, len(b.mode)+1+len(b.hash)+1+len(b.name))
	buf = append(buf, b.mode...)
	buf = append(buf, " "...)
	buf = append(buf, "blob"...)
	buf = append(buf, " "...)
	buf = append(buf, b.hash[:]...)
	buf = append(buf, " "...)
	buf = append(buf, []byte(b.name)...)
	return buf
}

func (t *treeObject) Bytes() []byte {
	var buf bytes.Buffer
	buf.Grow(t.contentLength() + 20)

	fmt.Fprintf(&buf, "tree %d\x00", t.contentLength())

	return buf.Bytes()
}

func (cmd *WriteTreeCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	gitRepo, err := findGitRepo()
	if err != nil {
		return err
	}

	indexFile := filepath.Join(gitRepo, ".git", "index")
	data, err := os.ReadFile(indexFile)
	if err != nil {
		return err
	}

	entries, err := readIndex(data)
	if err != nil {
		return err
	}

	treeObj := createTree(entries)

	fmt.Fprintf(w, "write-tree output: %v\n", treeObj)

	return nil
}

func createTree(entries []*indexEntry) *treeObject {
	tree := newTreeRoot()

	for _, entry := range entries {
		t := tree.getTree(entry.path)
		name := filepath.Base(entry.path)
		t.addBlob(entry.mode, entry.gitSha, name)
	}

	return tree
}

func (t *treeObject) getTree(path string) *treeObject {
	current := t

	parts := strings.Split(path, "/")
outer:
	for _, p := range parts[:len(parts)-1] {
		for _, ts := range current.trees {
			if ts.name == p {
				current = ts
				continue outer
			}
		}
		current = current.addTree(p)
	}

	return current
}

func newTreeRoot() *treeObject {
	return newTreeObject("")
}

func (t treeObject) contentLength() int {
	return 0
}

func (t *treeObject) String() string {
	var sb strings.Builder
	t.writeString(&sb, 0)
	return sb.String()
}

func (t *treeObject) writeString(sb *strings.Builder, depth int) {
	indent := strings.Repeat("  ", depth)

	for _, blob := range t.blobs {
		sb.WriteString(indent)
		sb.WriteString(blob.name)
		sb.WriteString("\n")
	}

	for _, tree := range t.trees {
		sb.WriteString(indent)
		sb.WriteString(tree.name)
		sb.WriteString("/\n")

		tree.writeString(sb, depth+1)
	}
}
