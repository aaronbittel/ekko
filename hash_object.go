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

	if cmd.useStdin {
		hash, err := HashObject(cmd.writeObject, os.Stdin)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, hex.EncodeToString(hash))
		return nil
	}

	if cmd.fs.NArg() == 0 {
		cmd.fs.Usage()
		fmt.Fprintln(cmd.fs.Output(), "to write hash an object either use '--stdin' or provide file(s)")
	}

	for i := range cmd.fs.NArg() {
		path := cmd.fs.Arg(i)
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open file %q: %w", path, err)
		}
		hash, err := HashObject(cmd.writeObject, f)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, hex.EncodeToString(hash))

		if err := f.Close(); err != nil {
			return fmt.Errorf("close file %q: %w", path, err)
		}
	}

	return nil
}

func HashObject(write bool, r io.Reader) (hash []byte, err error) {
	object, err := BlobFromFile(r)
	if err != nil {
		return nil, err
	}

	if write {
		hash, err = object.WriteToObjects()
		if err != nil {
			return nil, err
		}
	} else {
		hash, _ = object.Write(io.Discard)
	}

	return hash, nil
}

func runHashObject(r io.Reader, gitObjectType string, writeObject bool) (objectID []byte, err error) {
	objectData, err := buildObjectData(r, gitObjectType)
	if err != nil {
		return nil, err
	}

	hash := sha1.Sum(objectData)
	objectID = hash[:]

	// TODO: could make the other path (not writing) just take an io.Discard
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
