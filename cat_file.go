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
		expectedObjectKind Kind
		err                error
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
		expectedObjectKind, err = ParseKind(kindInput)
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

	objectPath, err := getObjectPath(gitRepo, hash)
	if err != nil {
		return err
	}

	f, err := os.Open(objectPath)
	if err != nil {
		return fmt.Errorf("open git object %q: %w", objectPath, err)
	}
	defer f.Close()

	object, err := ReadObject(f)
	if err != nil {
		return err
	}

	if cmd.requireObjectType() && expectedObjectKind != object.Kind {
		return fmt.Errorf("expected object kind %q, got %q", expectedObjectKind, object.Kind)
	}

	if cmd.showType {
		fmt.Fprintln(w, object.Kind.String())
		return nil
	}

	if cmd.prettyPrint && object.Kind == KindTree {
		treeNodes, err := parseTreeObjects(gitRepo, object)
		if err != nil {
			return err
		}

		for _, treeNode := range treeNodes {
			writeTreeNode(treeNode, w, false)
		}

		return nil
	}

	if _, err := io.CopyN(w, object.Reader, int64(object.ExpectedSize)); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

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
func (cfg CatFileCmd) requireObjectType() bool {
	return !cfg.prettyPrint && !cfg.showType
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

// Returns the next positional argument "" empty string, if none provided or arguments
// are exhausted
func (cmd *CatFileCmd) nextPositional() string {
	arg := cmd.fs.Arg(cmd.nextPositionalIdx)
	cmd.nextPositionalIdx++
	return arg
}
