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

type CatFileCmd struct {
	description string

	catFileConfig

	fs *flag.FlagSet
}

type catFileConfig struct {
	prettyPrint bool
	showType    bool
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
		cmd.fs.PrintDefaults()
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
		expectedType string
		objectName   string
	)

	if cmd.prettyPrint {
		objectName = cmd.fs.Arg(0)
		if objectName == "" {
			cmd.fs.Usage()
			return fmt.Errorf("missing object name")
		}
	} else if cmd.showType {
		objectName = cmd.fs.Arg(0)
		if objectName == "" {
			cmd.fs.Usage()
			return fmt.Errorf("missing object name")
		}
	} else {
		expectedType = cmd.fs.Arg(0)
		objectName = cmd.fs.Arg(1)
		if expectedType == "" || objectName == "" {
			cmd.fs.Usage()
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

	if cmd.showType {
		fmt.Fprintln(w, objectType)
		return nil
	}

	if !cmd.prettyPrint && objectType != expectedType {
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
