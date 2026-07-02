package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/aaronbittel/ekko/internal/objects"
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

	store := objects.NewStore(filepath.Join(gitRepo, ".git", "objects"))

	obj, err := store.Open(hash)
	if err != nil {
		return err
	}
	defer obj.Close()

	if obj.Kind != objects.KindTree {
		return fmt.Errorf("not a tree object")
	}

	treeEntries, err := obj.ParseAsTree()
	if err != nil {
		return err
	}

	for _, entry := range treeEntries {
		fmt.Fprintln(w, entry.Pretty())
	}

	return nil
}
