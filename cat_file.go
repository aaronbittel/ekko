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
	"strings"
)

var ErrBadFile = errors.New("bad file")

func catFile(fs *flag.FlagSet, w io.Writer, args ...string) error {
	fs.Usage = func() {
		fmt.Fprintf(w, "ekko-cat-file - Provide contents or details of repository objects\n\n")
		fs.PrintDefaults()
	}

	var (
		prettyPrint bool
		showType    bool
	)

	fs.BoolVar(&prettyPrint, "p", false, "Pretty-print the contents of <object> based on its type.")
	fs.BoolVar(&showType, "t", false, "Instead of the content, show the object type identified by <object>.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if prettyPrint && showType {
		return fmt.Errorf("-p and -t are mutually exclusive")
	}

	var (
		expectedType string
		objectName   string
	)

	if prettyPrint {
		objectName = fs.Arg(0)
		if objectName == "" {
			fs.Usage()
			return fmt.Errorf("missing object name")
		}
	} else if showType {
		objectName = fs.Arg(0)
		if objectName == "" {
			fs.Usage()
			return fmt.Errorf("missing object name")
		}
	} else {
		expectedType = fs.Arg(0)
		objectName = fs.Arg(1)
		if expectedType == "" || objectName == "" {
			fs.Usage()
			return fmt.Errorf("missing object name or expected object type")
		}
	}

	gitRepo, err := findGitRepo()
	if err != nil {
		return err
	}

	objectType, data, err := loadGitObject(gitRepo, objectName)
	if err != nil {
		return err
	}

	if showType {
		fmt.Fprintln(w, objectType)
		return nil
	}

	if !prettyPrint && objectType != expectedType {
		return fmt.Errorf("expected object type %q, got %q", expectedType, objectType)
	}

	content, err := runCatFile(objectType, data)
	if err != nil {
		return err
	}

	fmt.Fprint(w, content)

	return nil
}

func loadGitObject(gitRepo, objectName string) (objectType, content string, err error) {
	path, err := getObjectPath(gitRepo, objectName)
	if err != nil {
		return "", "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	content, err = decompressObject(f)
	if err != nil {
		return "", "", err
	}

	return getObjectType(content)
}

func runCatFile(objectType, content string) (string, error) {
	switch objectType {
	case typeBlob:
		return readBlob(content)
	case typeTree:
		fallthrough
	case typeTag:
		fallthrough
	case typeCommit:
		return "", fmt.Errorf("%s not implemented yet", objectType)
	default:
		panic("unreachable")
	}
}

func getObjectType(content string) (typ, rest string, err error) {
	typ, rest, found := strings.Cut(content, " ")
	if !found {
		return "", "", ErrBadFile
	}

	switch typ {
	case typeBlob, typeTree, typeTag, typeCommit:
		return typ, rest, nil
	default:
		return "", "", fmt.Errorf("unknown object info %q", typ)
	}
}

func decompressObject(r io.Reader) (string, error) {
	zr, err := zlib.NewReader(r)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	buf := new(bytes.Buffer)

	if _, err := io.Copy(buf, zr); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func readBlob(content string) (string, error) {
	sizeStr, rest, found := strings.Cut(content, "\x00")
	if !found {
		return "", ErrBadFile
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return "", ErrBadFile
	}

	if len(rest) != size {
		return "", ErrBadFile
	}

	return rest, nil
}
