package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
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

	var (
		r   io.Reader
		err error
	)

	if cmd.useStdin {
		r = os.Stdin
	} else {
		path := cmd.fs.Arg(0)
		if path == "" {
			return errors.New("no input file provided")
		}
		r, err = os.Open(path)
		if err != nil {
			return err
		}
	}

	objectID, err := runHashObject(r, cmd.gitObjectType, cmd.writeObject)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "%x\n", objectID)
	return nil
}

func runHashObject(r io.Reader, gitObjectType string, writeObject bool) (objectID []byte, err error) {
	objectData, err := buildObjectData(r, gitObjectType)
	if err != nil {
		return nil, err
	}

	hash := sha1.Sum(objectData)
	objectID = hash[:]

	if writeObject {
		objectIDStr := hex.EncodeToString(objectID)
		dirname := objectIDStr[:2]
		filename := objectIDStr[2:]

		dirpath := filepath.Join(".git", "objects", dirname)
		if err := os.Mkdir(dirpath, 0777); err != nil {
			return nil, err
		}

		path := filepath.Join(dirpath, filename)

		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		zr := zlib.NewWriter(f)
		defer zr.Close()
		zr.Write(objectData)
	}

	return objectID, nil
}

func computeObjectHash(path, objectType string, r io.Reader) (gitSha1, error) {
	data, err := buildObjectData(r, objectType)
	if err != nil {
		return gitSha1{}, err
	}

	return sha1.Sum(data), nil
}

func buildObjectData(r io.Reader, objectType string) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	switch objectType {
	case typeBlob:
		fmt.Fprintf(&buf, "blob %d\x00", len(data))
		buf.Write(data)
	case typeCommit, typeTree, typeTag:
		return nil, fmt.Errorf("object type %q is not supported yet", objectType)
	default:
		return nil, fmt.Errorf("invalid object type %q", objectType)
	}

	return buf.Bytes(), nil
}
