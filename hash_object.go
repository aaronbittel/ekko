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

func runHashObject(r io.Reader, gitObjType string, writeObj bool) (objID []byte, err error) {
	objData, err := buildObjData(r, gitObjType)
	if err != nil {
		return nil, err
	}

	hash := sha1.Sum(objData)
	objID = hash[:]

	if writeObj {
		objIDStr := hex.EncodeToString(objID)
		dirname := objIDStr[:2]
		filename := objIDStr[2:]

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
		zr.Write(objData)
	}

	return objID, nil
}

func hashObject(fs *flag.FlagSet, w io.Writer, args ...string) error {
	fs.Usage = func() {
		fmt.Fprintf(w, "ekko-hash-object - Compute object ID and optionally create an object from a file\n\n")
		fs.PrintDefaults()
	}

	var (
		gitObjType = typeBlob
		useStdin   bool
		writeObj   bool
	)

	fs.Func("t", "Specify the type of object to be created (default: \"blob\"). Possible values are commit, tree, blob, and tag.", func(typ string) error {
		validTypes := []string{typeBlob, typeCommit, typeTree, typeTag}
		if idx := slices.Index(validTypes, typ); idx == -1 {
			return fmt.Errorf("invalid object type %q", typ)
		} else {
			gitObjType = validTypes[idx]
			return nil
		}
	})
	fs.BoolVar(&useStdin, "stdin", false, "Read the object from standard input instead of from a file.")
	fs.BoolVar(&writeObj, "w", false, "Actually write the object into the object database.")

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

	objID, err := runHashObject(r, gitObjType, writeObj)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "%x\n", objID)
	return nil
}

func buildObjData(r io.Reader, objType string) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	switch objType {
	case typeBlob:
		fmt.Fprintf(&buf, "blob %d\x00", len(data))
		buf.Write(data)
	case typeCommit, typeTree, typeTag:
		return nil, fmt.Errorf("object type %q is not supported yet", objType)
	default:
		return nil, fmt.Errorf("invalid object type %q", objType)
	}

	return buf.Bytes(), nil
}
