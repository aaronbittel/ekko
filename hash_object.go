package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/aaronbittel/ekko/internal/objects"
)

const (
	typeBlob   = "blob"
	typeCommit = "commit"
	typeTree   = "tree"
	typeTag    = "tag"
)

type HashObjectCmd struct {
	description string

	hashObjectConfig

	fs *flag.FlagSet
}

type hashObjectConfig struct {
	gitObjectType string
	useStdin      bool
	writeObject   bool
}

func NewHashObjectCmd(fs *flag.FlagSet) *HashObjectCmd {
	cmd := &HashObjectCmd{
		description: "Hash and optionally store objects",
		fs:          fs,
	}

	cmd.hashObjectConfig = hashObjectConfig{gitObjectType: typeBlob}

	cmd.defineFlags()
	return cmd
}

func (cmd *HashObjectCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-hash-object - Compute object ID and optionally create an object from a file\n\n")
		cmd.fs.PrintDefaults()
		fmt.Fprintln(cmd.fs.Output())
	}

	cmd.fs.Func("t", "Specify the type of object to be created (default: \"blob\"). Possible values are commit, tree, blob, and tag.", func(typ string) error {
		validTypes := []string{typeBlob, typeCommit, typeTree, typeTag}
		if idx := slices.Index(validTypes, typ); idx == -1 {
			return fmt.Errorf("invalid object type %q", typ)
		} else {
			cmd.gitObjectType = validTypes[idx]
			return nil
		}
	})
	cmd.fs.BoolVar(&cmd.useStdin, "stdin", false, "Read the object from standard input instead of from a file.")
	cmd.fs.BoolVar(&cmd.writeObject, "w", false, "Actually write the object into the object database.")
}

func (cmd *HashObjectCmd) Description() string {
	return cmd.description
}

func (cmd *HashObjectCmd) Name() string {
	return cmd.fs.Name()
}

func (cmd *HashObjectCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	gitRepo, err := findGitRepo()
	if err != nil {
		return err
	}

	store := objects.NewStore(filepath.Join(gitRepo, ".git", "objects"))

	if cmd.useStdin {
		hash, err := cmd.process(store, os.Stdin)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, hash)
		return nil
	}

	if cmd.fs.NArg() == 0 {
		cmd.fs.Usage()
		return errors.New("use --stdin or provide file(s)")
	}

	for i := range cmd.fs.NArg() {
		// Use an anonymous function so defer closes each file at the end of its
		// iteration, rather than at the end of the enclosing function.
		err := func() error {
			path := cmd.fs.Arg(i)
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("open file %q: %w", path, err)
			}
			defer f.Close()

			hash, err := cmd.process(store, f)
			if err != nil {
				return err
			}

			fmt.Fprintln(w, hash)
			return nil
		}()

		if err != nil {
			return err
		}
	}

	return nil
}

func (cmd *HashObjectCmd) process(store objects.Store, r io.Reader) (hash string, err error) {
	obj, err := objects.BlobFromReader(r)
	if err != nil {
		return "", err
	}
	defer obj.Close()

	if cmd.writeObject {
		hash, err = store.Write(obj)
		if err != nil {
			return "", err
		}
	} else {
		hashBytes, err := obj.Write(io.Discard)
		if err != nil {
			return "", err
		}
		hash = hex.EncodeToString(hashBytes)
	}

	return hash, nil
}
