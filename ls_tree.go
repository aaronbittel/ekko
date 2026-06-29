package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	treeHashHex := cmd.fs.Arg(0)
	if treeHashHex == "" {
		cmd.fs.Usage()
		return fmt.Errorf("missing tree hash argument")
	}

	gitRepo, err := findGitRepo()
	if err != nil {
		return err
	}

	objectPath := filepath.Join(gitRepo, ".git", "objects", treeHashHex[:2], treeHashHex[2:])
	f, err := os.Open(objectPath)
	if err != nil {
		return err
	}
	defer f.Close()

	LsTree(w, treeHashHex, cmd.nameOnly)

	return nil
}

func LsTree(w io.Writer, treeHashHex string, nameOnly bool) error {
	object, err := Read(treeHashHex)
	if err != nil {
		return err
	}

	return lsTreeImpl(w, object, nameOnly)
}

func lsTreeImpl(w io.Writer, object *Object[*bufio.Reader], nameOnly bool) error {
	for {
		modeAndName, err := object.Reader.ReadSlice(0)
		if err != nil {
			return err
		}
		object.ExptecedSize -= uint64(len(modeAndName))
		if object.ExptecedSize <= 0 {
			return fmt.Errorf("read tree entry")
		}
		modeAndName = modeAndName[:len(modeAndName)-1]
		mode, name, found := bytes.Cut(modeAndName, []byte{' '})
		if !found {
			return fmt.Errorf("malformed tree entry, missing ' '")
		}

		hashBuf := make([]byte, 20)
		n, err := io.ReadFull(object.Reader, hashBuf)
		if err != nil {
			return fmt.Errorf("read tree entry hash")
		}
		object.ExptecedSize -= uint64(n)
		if object.ExptecedSize < 0 {
			return fmt.Errorf("read tree entry")
		}

		hashHex := hex.EncodeToString(hashBuf)

		if nameOnly {
			fmt.Fprintf(w, "%s\n", name)
		} else {
			entryObject, err := Read(hashHex)
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "%06s %s %s    %s\n", mode, entryObject.Kind, hashHex, name)
		}

		if object.ExptecedSize == 0 {
			break
		}
	}

	return nil
}
