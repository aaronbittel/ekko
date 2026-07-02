package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/aaronbittel/ekko/internal/objects"
)

type CommitTreeCmd struct {
	description string

	commitTreeConfig

	fs *flag.FlagSet
}

type commitTreeConfig struct {
	message string
}

func NewCommitTreeCmd(fs *flag.FlagSet) *CommitTreeCmd {
	cmd := &CommitTreeCmd{
		description: "Create a new commit object",
		fs:          fs,
	}

	cmd.defineFlags()
	return cmd
}

func (cmd *CommitTreeCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-commit-tree - Create a new commit object\n\n")
		fmt.Fprintf(cmd.fs.Output(), "usage: ekko commit-tree <tree>\n\n")
		fmt.Fprintln(cmd.fs.Output(), "options:")
		cmd.fs.PrintDefaults()
		fmt.Fprintln(cmd.fs.Output())
	}

	cmd.fs.StringVar(&cmd.message, "m", "", "A paragraph in the commit log message.")
}

func (cmd *CommitTreeCmd) Description() string {
	return cmd.description
}

func (cmd *CommitTreeCmd) Name() string {
	return cmd.fs.Name()
}

func (cmd *CommitTreeCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	if cmd.message == "" {
		return fmt.Errorf("missing required flag: -m <message>")
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

	objKind, err := store.Kind(hash)
	if err != nil {
		return err
	}

	if objKind != objects.KindTree {
		return fmt.Errorf("not a tree object: %q", objKind)
	}

	treePath, err := store.GetObjectPath(hash)
	if err != nil {
		return err
	}

	treeHash, err := objects.GitHashFromObjectPath(treePath)
	if err != nil {
		return err
	}

	commitObj := objects.NewCommit(treeHash, nil, "Bob Doe", "bob.doe@example.com", "first commit")
	commitHash, err := store.WriteCommit(commitObj)
	if err != nil {
		return err
	}

	fmt.Fprintln(w, commitHash)

	return nil
}
