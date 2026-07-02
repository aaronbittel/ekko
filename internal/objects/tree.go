package objects

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	ModeRegularFile    string = "100644"
	ModeExecutableFile string = "100755"
	ModeSymlink        string = "120000"
	ModeTree           string = "40000"
)

type Tree struct {
	Entries []TreeEntry
}

func (t Tree) Encode() ([]byte, error) {
	var buf bytes.Buffer

	for _, entry := range t.Entries {
		b, err := entry.Encode()
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}

	return buf.Bytes(), nil
}

type TreeEntry struct {
	Hash string
	Name string
	Mode string
	Kind Kind
}

func ParseTreeObject(r io.Reader) ([]TreeEntry, error) {
	obj, err := NewReader(io.NopCloser(r))
	if err != nil {
		return nil, err
	}
	defer obj.Close()

	return obj.ParseAsTree()
}

func (o *Object) ParseAsTree() ([]TreeEntry, error) {
	if o.Kind != KindTree {
		return nil, fmt.Errorf("provided object is not a 'tree' but a %q", o.Kind)
	}

	treeEntries := []TreeEntry{}

	for {
		treeEntry, done, err := o.parseTreeEntry()
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
		treeEntries = append(treeEntries, treeEntry)
	}

	return treeEntries, nil
}

func (o *Object) parseTreeEntry() (treeEntry TreeEntry, done bool, err error) {
	if _, err := o.Peek(1); err != nil && errors.Is(err, io.EOF) {
		return TreeEntry{}, true, nil
	}

	modeAndName, err := o.ReadString(0)
	if err != nil {
		return TreeEntry{}, false, err
	}
	modeAndName = modeAndName[:len(modeAndName)-1] // remove \x00

	mode, name, found := strings.Cut(modeAndName, " ")
	if !found {
		return TreeEntry{}, false, errors.New("invalid tree entry")
	}

	if !validTreeMode(mode) {
		return TreeEntry{}, false, fmt.Errorf("invalid tree mode: %q", mode)
	}

	kind := mustDeriveKindFromMode(mode)

	hashBytes := make([]byte, 20)
	if _, err := io.ReadFull(o, hashBytes); err != nil {
		return TreeEntry{}, false, err
	}

	return TreeEntry{
		Hash: hex.EncodeToString(hashBytes),
		Name: name,
		Mode: mode,
		Kind: kind,
	}, false, nil
}

func mustDeriveKindFromMode(mode string) Kind {
	switch mode {
	case ModeRegularFile, ModeExecutableFile, ModeSymlink:
		return KindBlob
	case ModeTree:
		return KindTree
	default:
		panic(fmt.Sprintf("illegal mode: %q", mode))
	}
}

func validTreeMode(mode string) bool {
	switch mode {
	case ModeRegularFile, ModeExecutableFile, ModeTree, ModeSymlink:
		return true
	default:
		return false
	}
}

func (e TreeEntry) Pretty() string {
	return fmt.Sprintf("%06s %s %s\t%s", e.Mode, e.Kind, e.Hash, e.Name)
}

func (e TreeEntry) Encode() ([]byte, error) {
	hashBytes, err := hex.DecodeString(e.Hash)
	if err != nil {
		return nil, err
	}
	buf := fmt.Appendf(nil, "%s %s\x00", e.Mode, e.Name)
	buf = append(buf, hashBytes...)
	return buf, nil
}
