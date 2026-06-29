package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type Kind int

const (
	KindBlob Kind = iota
	KindTree
	KindCommit
	KindTag
)

type Object[R io.Reader] struct {
	Kind         Kind
	ExpectedSize uint64
	Reader       R
}

func ReadObject(r io.Reader) (*Object[*bufio.Reader], error) {
	zr, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}

	bzr := bufio.NewReader(zr)

	header, err := bzr.ReadSlice(0)
	if err != nil {
		return nil, fmt.Errorf("invalid git object header: %w", err)
	}
	header = header[:len(header)-1] // remove trailing \x00

	kindBytes, sizeBytes, found := bytes.Cut(header, []byte{' '})
	if !found {
		return nil, fmt.Errorf("invalid git object header: missing space separator")
	}

	var kind Kind

	switch {
	case bytes.Equal(kindBytes, []byte("blob")):
		kind = KindBlob
	case bytes.Equal(kindBytes, []byte("tree")):
		kind = KindTree
	case bytes.Equal(kindBytes, []byte("commit")):
		kind = KindCommit
	case bytes.Equal(kindBytes, []byte("tag")):
		kind = KindTag
	default:
		return nil, fmt.Errorf("unknown git object type %q", kindBytes)
	}

	size, err := strconv.ParseUint(string(sizeBytes), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("header has invalid size %d: %w", size, err)
	}

	return &Object[*bufio.Reader]{
		Kind:         kind,
		ExpectedSize: size,
		Reader:       bzr,
	}, nil
}

func (o *Object[R]) Write(w io.Writer) (hash []byte, err error) {
	zw := zlib.NewWriter(w)
	hw := &ObjectHashWriter{w: zw, hash: sha1.New()}
	fmt.Fprintf(hw, "%s %d\x00", o.Kind, o.ExpectedSize)
	if _, err := io.CopyN(hw, o.Reader, int64(o.ExpectedSize)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return hw.Sum(), nil
}

func (o *Object[R]) WriteToObjects() (hash []byte, err error) {
	tmp, err := os.CreateTemp(".", "writeObjects-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer tmp.Close()

	hash, err = o.Write(tmp)
	if err != nil {
		return nil, err
	}

	hashHex := hex.EncodeToString(hash)

	gitRepo, err := findGitRepo()
	if err != nil {
		return nil, err
	}

	objectPath := filepath.Join(gitRepo, ".git", "objects", hashHex[:2], hashHex[2:])
	objectDir := filepath.Dir(objectPath)
	if err := os.MkdirAll(objectDir, 0o755); err != nil {
		return nil, fmt.Errorf("create object dir %q: %w", objectDir, err)
	}

	if err := os.Rename(tmp.Name(), objectPath); err != nil {
		return nil, fmt.Errorf("rename tmp file %q: %w", tmp.Name(), err)
	}

	return hash, nil
}

func BlobFromFile(r io.Reader) (*Object[io.Reader], error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return &Object[io.Reader]{
		Kind:         KindBlob,
		ExpectedSize: uint64(len(data)),
		Reader:       bytes.NewReader(data),
	}, nil
}

type ObjectHashWriter struct {
	w    io.Writer
	hash hash.Hash
}

func (hw *ObjectHashWriter) Write(b []byte) (n int, err error) {
	n, err = hw.w.Write(b)
	if err != nil {
		return 0, err
	}
	hw.hash.Write(b[:n])
	return n, nil
}

func (hw *ObjectHashWriter) Sum() []byte {
	return hw.hash.Sum(nil)
}

func (k Kind) String() string {
	switch k {
	case KindBlob:
		return "blob"
	case KindTree:
		return "tree"
	case KindCommit:
		return "commit"
	case KindTag:
		return "tag"
	default:
		panic("unknown object kind")
	}
}

func ParseKind(arg string) (Kind, error) {
	switch arg {
	case "blob":
		return KindBlob, nil
	case "tree":
		return KindTree, nil
	case "commit":
		return KindCommit, nil
	case "tag":
		return KindTag, nil
	default:
		return 0, fmt.Errorf("unknown object kind %q", arg)
	}
}
