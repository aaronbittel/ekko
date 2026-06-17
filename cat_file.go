package main

import (
	"bytes"
	"compress/zlib"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

var (
	ErrBadFile = errors.New("bad file")

	blobStart = []byte("blob ")
	nullByte  = []byte{0x00}
)

func catFile(fs *flag.FlagSet, w io.Writer, args ...string) error {
	fs.Usage = func() {
		fmt.Fprintf(w, "ekko-cat-file - Provide contents or details of repository objects")
		fs.PrintDefaults()
	}

	var (
		prettyPrint bool
	)

	fs.BoolVar(&prettyPrint, "p", false, "Pretty-print the contents of <object> based on its type.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	var (
		content []byte
		err     error
	)

	if prettyPrint {
		object := fs.Arg(0)
		_ = object
		return fmt.Errorf("\"-p\" is not implemented yet")
	} else {
		objectType := fs.Arg(0)
		object := fs.Arg(1)

		content, err = runCatFile(objectType, object)
		if err != nil {
			return fmt.Errorf("ekko cat-file: %s: %v", object, err)
		}
	}

	fmt.Fprint(w, string(content))

	return nil
}

func runCatFile(objectType, objectID string) ([]byte, error) {
	path, err := getObjectPath(objectID)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := readObjectData(objectID, f)
	if err != nil {
		return nil, err
	}

	return parseObject(objectType, data)
}

func readObjectData(objectID string, r io.Reader) ([]byte, error) {
	zr, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)

	io.Copy(buf, zr)
	if err := zr.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func parseObject(objectType string, data []byte) ([]byte, error) {
	switch objectType {
	case typeBlob:
		if content, err := readBlob(data); err != nil {
			return content, err
		} else {
			return content, nil
		}
	case typeCommit, typeTree, typeTag:
		return nil, fmt.Errorf("type %q is not supported yet", objectType)
	default:
		return nil, fmt.Errorf("invalid object type %q", objectType)
	}
}

func readBlob(buf []byte) ([]byte, error) {
	if len(buf) < len(blobStart) || !bytes.Equal(buf[:len(blobStart)], blobStart) {
		return nil, ErrBadFile
	}
	buf = buf[len(blobStart):]

	idx := bytes.Index(buf, nullByte)
	if idx == -1 {
		return nil, ErrBadFile
	}

	size, err := strconv.Atoi(string(buf[:idx]))
	if err != nil {
		return nil, ErrBadFile
	}

	buf = buf[idx+1:]

	if len(buf) != size {
		return nil, ErrBadFile
	}

	return buf, nil
}
