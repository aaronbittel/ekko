package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
)

type LsTreeCmd struct {
	description string

	lsTreeConfig

	fs *flag.FlagSet
}

type lsTreeConfig struct {
	nameOnly bool
}

func NewLsTreeCmd(fs *flag.FlagSet) *LsTreeCmd {
	cmd := &LsTreeCmd{
		description: "List the contents of a tree object",
		fs:          fs,
	}

	cmd.defineFlags()
	return cmd
}

func (cmd *LsTreeCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-ls-tree - List the contents of a tree object")
		cmd.fs.PrintDefaults()
	}

	cmd.fs.BoolVar(&cmd.nameOnly, "name-only", false, "List only filenames (instead of the \"long\" output), one per line.")
}

func (cmd *LsTreeCmd) Description() string {
	return cmd.description
}

func (cmd *LsTreeCmd) Name() string {
	return cmd.fs.Name()
}

func (cmd *LsTreeCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	hash := cmd.fs.Arg(0)
	if hash == "" {
		cmd.fs.Usage()
		return fmt.Errorf("missing tree hash argument")
	}

	gitRepo, err := findGitRepo()
	if err != nil {
		return err
	}

	objectPath, err := getObjectPath(gitRepo, hash)
	if err != nil {
		return err
	}

	f, err := os.Open(objectPath)
	if err != nil {
		return fmt.Errorf("open git object %q: %w", objectPath, err)
	}
	defer f.Close()

	object, err := ReadObject(f)
	if err != nil {
		return err
	}
	if object.Kind != KindTree {
		return fmt.Errorf("not a tree object")
	}

	treeObjects, err := parseTreeObjects(gitRepo, object)
	if err != nil {
		return err
	}

	for _, treeObj := range treeObjects {
		writeTreeObject(treeObj, w, cmd.nameOnly)
	}

	return nil
}

type lsTreeObject struct {
	hash string
	name []byte
	mode []byte
	kind Kind
}

func parseTreeObjects(gitRepo string, object *Object[*bufio.Reader]) ([]lsTreeObject, error) {
	treeObjects := []lsTreeObject{}

	for object.ExpectedSize > 0 {
		treeObj, err := parseTreeObject(gitRepo, object)
		if err != nil {
			return nil, err
		}
		treeObjects = append(treeObjects, treeObj)
	}

	return treeObjects, nil
}

func parseTreeObject(gitRepo string, object *Object[*bufio.Reader]) (lsTreeObject, error) {
	modeAndName, err := object.Reader.ReadSlice(0)
	if err != nil {
		return lsTreeObject{}, err
	}
	if object.ExpectedSize < uint64(len(modeAndName)) {
		return lsTreeObject{}, fmt.Errorf("read tree entry")
	}
	object.ExpectedSize -= uint64(len(modeAndName))
	modeAndName = modeAndName[:len(modeAndName)-1]
	mode, name, found := bytes.Cut(modeAndName, []byte{' '})
	if !found {
		return lsTreeObject{}, fmt.Errorf("malformed tree entry, missing ' '")
	}

	hashBuf := make([]byte, 20)
	n, err := io.ReadFull(object.Reader, hashBuf)
	if err != nil {
		return lsTreeObject{}, fmt.Errorf("read tree entry hash")
	}
	if object.ExpectedSize < uint64(n) {
		return lsTreeObject{}, fmt.Errorf("tree entry exceeds expected size")
	}
	object.ExpectedSize -= uint64(n)

	hash := hex.EncodeToString(hashBuf)

	objectPath, err := getObjectPath(gitRepo, hash)
	if err != nil {
		return lsTreeObject{}, err
	}

	f, err := os.Open(objectPath)
	if err != nil {
		return lsTreeObject{}, err
	}
	defer f.Close()

	entryObject, err := ReadObject(f)
	if err != nil {
		return lsTreeObject{}, err
	}

	return lsTreeObject{
		hash: hash,
		name: name,
		mode: mode,
		kind: entryObject.Kind,
	}, nil
}

func writeTreeObject(obj lsTreeObject, w io.Writer, nameOnly bool) {
	if nameOnly {
		fmt.Fprintf(w, "%s\n", obj.name)
	} else {
		fmt.Fprintf(w, "%06s %s %s\t%s\n", obj.mode, obj.kind, obj.hash, obj.name)
	}
}
