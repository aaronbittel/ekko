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

		filepath := filepath.Join(dirpath, filename)

		f, err := os.Create(filepath)
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

func hashObject(fs *flag.FlagSet, w io.Writer, args ...string) error {
	fs.Usage = func() {
		fmt.Fprintf(w, "ekko-hash-object - Compute object ID and optionally create an object from a file\n\n")
		fs.PrintDefaults()
	}

	var (
		gitObjectType = typeBlob
		useStdin      bool
		writeObject   bool
	)

	fs.Func("t", "Specify the type of object to be created (default: \"blob\"). Possible values are commit, tree, blob, and tag.", func(typ string) error {
		validTypes := []string{typeBlob, typeCommit, typeTree, typeTag}
		if idx := slices.Index(validTypes, typ); idx == -1 {
			return fmt.Errorf("invalid object type %q", typ)
		} else {
			gitObjectType = validTypes[idx]
			return nil
		}
	})
	fs.BoolVar(&useStdin, "stdin", false, "Read the object from standard input instead of from a file.")
	fs.BoolVar(&writeObject, "w", false, "Actually write the object into the object database.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	var (
		r   io.Reader
		err error
	)

	if useStdin {
		r = os.Stdin
	} else {
		filepath := fs.Arg(0)
		if filepath == "" {
			return errors.New("no input file provided")
		}
		r, err = os.Open(filepath)
		if err != nil {
			return err
		}
	}

	objectID, err := runHashObject(r, gitObjectType, writeObject)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "%x\n", objectID)
	return nil
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
