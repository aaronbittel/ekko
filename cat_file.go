package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/aaronbittel/ekko/internal/objects"
)

var ErrBadFile = errors.New("bad file")

type CatFileCmd struct {
	description string

	catFileConfig

	fs *flag.FlagSet
}

type catFileConfig struct {
	prettyPrint bool
	showType    bool

	nextPositionalIdx int
}

func NewCatFileCmd(fs *flag.FlagSet) *CatFileCmd {
	cmd := &CatFileCmd{
		description: "Inspect Git objects from the database",
		fs:          fs,
	}

	cmd.defineFlags()
	return cmd
}

func (cmd *CatFileCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-cat-file - Provide contents or details of repository objects\n\n")
		fmt.Fprintf(cmd.fs.Output(), "Usage: ekko cat-file <type> <object>\n\n")
		cmd.fs.PrintDefaults()
		fmt.Fprintln(cmd.fs.Output())
	}

	cmd.fs.BoolVar(&cmd.prettyPrint, "p", false, "Pretty-print the contents of <object> based on its type.")
	cmd.fs.BoolVar(&cmd.showType, "t", false, "Instead of the content, show the object type identified by <object>.")

}

func (cmd *CatFileCmd) Description() string {
	return cmd.description
}

func (cmd *CatFileCmd) Name() string {
	return cmd.fs.Name()
}

func (cmd *CatFileCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	if cmd.prettyPrint && cmd.showType {
		return fmt.Errorf("-p and -t are mutually exclusive")
	}

	var (
		expectedObjKind objects.Kind
		err             error
	)

	if cmd.requireObjectType() {
		if cmd.fs.NArg() != 2 {
			cmd.fs.Usage()
			return ErrCliUsageError
		}

		kindInput := cmd.nextPositional()
		if kindInput == "" {
			cmd.fs.Usage()
			return fmt.Errorf("object kind must be provided when no '-p' and no '-t' flags are used")
		}
		expectedObjKind, err = objects.ParseKind(kindInput)
		if err != nil {
			return err
		}
	}

	hash := cmd.nextPositional()
	if hash == "" {
		cmd.fs.Usage()
		return fmt.Errorf("missing object name")
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

	if cmd.showType {
		fmt.Fprintln(w, obj.Kind)
		return nil
	}

	if cmd.requireObjectType() && expectedObjKind != obj.Kind {
		return fmt.Errorf("expected object kind %q, got %q", expectedObjKind, obj.Kind)
	}

	if cmd.prettyPrint && obj.Kind == objects.KindTree {
		treeEntries, err := obj.ParseAsTree()
		if err != nil {
			return err
		}

		for _, entry := range treeEntries {
			fmt.Fprintln(w, entry.Pretty())
		}

		return nil
	}

	if _, err := obj.WriteTo(w); err != nil {
		return err
	}

	return nil
}

func (cfg CatFileCmd) requireObjectType() bool {
	return !cfg.prettyPrint && !cfg.showType
}

// Returns the next positional argument "" empty string, if none provided or arguments
// are exhausted
func (cmd *CatFileCmd) nextPositional() string {
	arg := cmd.fs.Arg(cmd.nextPositionalIdx)
	cmd.nextPositionalIdx++
	return arg
}
