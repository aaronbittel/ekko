package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
	hash, err := writeTree(treeObj, FileStorage{objectPath: filepath.Join(gitRepo, ".git", "objects")})
	if err != nil {
		return err
	}

	fmt.Fprint(w, hex.EncodeToString(hash[:]))

	return nil
}

type ObjectStore interface {
	Put(hash gitSha1, data []byte) error
}

type treeObject struct {
	name  string
	blobs []*blobEntry
	trees []*treeObject
}

const (
	EntryRegularFile    string = "100644"
	EntryExecutableFile string = "100755"
	EntryTree           string = "40000"
)

type treeEntry struct {
	typ  string
	name string
	hash gitSha1
}

func (t treeEntry) encode() []byte {
	buf := make([]byte, 0, 1024)
	buf = append(buf, t.typ...)
	buf = append(buf, ' ')
	buf = append(buf, t.name...)
	buf = append(buf, 0)
	buf = append(buf, t.hash[:]...)
	return buf
}

func encodeTree(name string, hash gitSha1) []byte {
	buf := make([]byte, 0, 5+1+len(name)+1+len(hash))
	buf = append(buf, "40000"...)
	buf = append(buf, ' ')
	buf = append(buf, name...)
	buf = append(buf, 0)
	buf = append(buf, hash[:]...)
	return buf
}

func writeTree(treeObj *treeObject, store ObjectStore) (gitSha1, error) {
	blobEntry := func(blob *blobEntry) treeEntry {
		typ := EntryRegularFile
		if bytes.Equal(blob.mode, blobModeExecutable) {
			typ = EntryExecutableFile
		}
		return treeEntry{typ: typ, hash: blob.hash, name: blob.name}
	}

	var (
		blobIdx, treeIdx int
		entries          []treeEntry
	)

	for blobIdx < len(treeObj.blobs) || treeIdx < len(treeObj.trees) {
		if blobIdx >= len(treeObj.blobs) {
			// write all remaining trees
			for ; treeIdx < len(treeObj.trees); treeIdx++ {
				hash, err := writeTree(treeObj.trees[treeIdx], store)
				if err != nil {
					return gitSha1{}, err
				}
				entries = append(entries, treeEntry{typ: EntryTree, name: treeObj.trees[treeIdx].name, hash: hash})
			}
			break
		}

		if treeIdx >= len(treeObj.trees) {
			// write all remaining blobs
			for ; blobIdx < len(treeObj.blobs); blobIdx++ {
				entries = append(entries, blobEntry(treeObj.blobs[blobIdx]))
			}
			break
		}

		if treeObj.blobs[blobIdx].name < treeObj.trees[treeIdx].name {
			// add blob
			entries = append(entries, blobEntry(treeObj.blobs[blobIdx]))
			blobIdx += 1
		} else {
			// write tree
			hash, err := writeTree(treeObj.trees[treeIdx], store)
			if err != nil {
				return gitSha1{}, err
			}
			entries = append(entries, treeEntry{typ: EntryTree, name: treeObj.trees[treeIdx].name, hash: hash})
			treeIdx += 1
		}
	}

	buf := serializeTree(entries)
	hash := sha1.Sum(buf)

	if err := store.Put(hash, buf); err != nil {
		return gitSha1{}, err
	}

	return hash, nil
}

func serializeTree(entries []treeEntry) []byte {
	buf := make([]byte, 0, 1024)

	for _, entry := range entries {
		buf = append(buf, entry.encode()...)
	}

	obj := make([]byte, 0, 1024)
	obj = append(obj, "tree"...)
	obj = append(obj, ' ')
	obj = strconv.AppendInt(obj, int64(len(buf)), 10)
	obj = append(obj, 0)
	obj = append(obj, buf...)

	return obj
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

func (b blobEntry) encode() []byte {
	buf := make([]byte, 0, len(b.mode)+1+len(b.name)+1+len(b.hash))
	buf = append(buf, b.mode...)
	buf = append(buf, ' ')
	buf = append(buf, []byte(b.name)...)
	buf = append(buf, 0)
	buf = append(buf, b.hash[:]...)
	return buf
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

type FileStorage struct {
	objectPath string
}

func (fs FileStorage) Put(hash gitSha1, data []byte) error {
	encoded := hex.EncodeToString(hash[:])
	dir := encoded[:2]
	name := encoded[2:]

	dirPath := filepath.Join(fs.objectPath, dir)
	if err := os.Mkdir(dirPath, 0755); err != nil {
		return err
	}

	objPath := filepath.Join(dirPath, name)

	f, err := os.Create(objPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zlib.NewWriter(f)
	if _, err := zw.Write(data); err != nil {
		return err
	}
	defer zw.Close()

	return nil
}
