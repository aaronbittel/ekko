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
	"strconv"
	"time"
)

type CommitTreeCmd struct {
	description string

	commitTreeConfig

	fs *flag.FlagSet
}

type commitTreeConfig struct {
	message string
}

func NewCommitTreeCmd(fs *flag.FlagSet) *CommitTreeCmd {
	cmd := &CommitTreeCmd{
		description: "Create a new commit object",
		fs:          fs,
	}

	cmd.defineFlags()
	return cmd
}

func (cmd *CommitTreeCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-commit-tree - Create a new commit object\n\n")
		fmt.Fprintf(cmd.fs.Output(), "usage: ekko commit-tree <tree>\n\n")
		fmt.Fprintln(cmd.fs.Output(), "options:")
		cmd.fs.PrintDefaults()
		fmt.Fprintln(cmd.fs.Output())
	}

	cmd.fs.StringVar(&cmd.message, "m", "", "A paragraph in the commit log message.")
}

func (cmd *CommitTreeCmd) Description() string {
	return cmd.description
}

func (cmd *CommitTreeCmd) Name() string {
	return cmd.fs.Name()
}

func (cmd *CommitTreeCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	treeHashInput := cmd.fs.Arg(0)
	if treeHashInput == "" {
		cmd.fs.Usage()
		return fmt.Errorf("missing tree hash argument")
	}

	gitRepo, err := findGitRepo()
	if err != nil {
		return err
	}

	treePath, err := getObjectPath(gitRepo, treeHashInput)
	if err != nil {
		return err
	}

	if cmd.message == "" {
		return fmt.Errorf("missing required flag: -m <message>")
	}

	dir := filepath.Base(filepath.Dir(treePath))
	file := filepath.Base(treePath)

	user := user{name: "Bob Doe", email: "bob.doe@example.com"}
	commitObject := newCommitObject(dir+file, nil, user, user, "first commit")
	data := commitObject.Encode()

	commitHash := sha1.Sum(data)
	commitHex := hex.EncodeToString(commitHash[:])

	commitDir := commitHex[:2]
	commitName := commitHex[2:]

	commitDirPath := filepath.Join(gitRepo, ".git", "objects", commitDir)

	if err := os.Mkdir(commitDirPath, 0755); err != nil {
		return err
	}

	commitFilePath := filepath.Join(commitDirPath, commitName)
	f, err := os.Create(commitFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zlib.NewWriter(f)
	defer zw.Close()
	if _, err := zw.Write(data); err != nil {
		return err
	}

	fmt.Fprintln(w, commitHex)

	return nil
}

type commitObject struct {
	treeHashHex  string
	parentHashes []gitSha1
	author       signature
	committer    signature
	message      string
}

func newCommitObject(treeHashHex string, parentHashes []gitSha1, author, committer user, message string) *commitObject {
	now := time.Now()
	secondsSinceEpoch := now.Unix()

	_, offset := now.Zone()
	hours := offset / 3600
	mins := (offset % 3600) / 60
	tz := fmt.Sprintf("%+03d%02d", hours, mins)

	timezone := timestamp{seconds: secondsSinceEpoch, offset: tz}

	return &commitObject{
		treeHashHex:  treeHashHex,
		parentHashes: parentHashes,
		author:       signature{role: roleAuthor, user: author, timestamp: timezone},
		committer:    signature{role: roleCommitter, user: committer, timestamp: timezone},
		message:      message,
	}
}

type signature struct {
	role commitRole
	user
	timestamp
}

type timestamp struct {
	seconds int64
	offset  string
}

//lint:ignore U1000 keep constructor for now
func newSignature(role commitRole, name, email string, secondsSinceEpoch int64, timezone string) *signature {
	return &signature{
		role:      role,
		user:      user{name: name, email: email},
		timestamp: timestamp{seconds: secondsSinceEpoch, offset: timezone},
	}
}

type commitRole string

const (
	roleAuthor    commitRole = "author"
	roleCommitter commitRole = "committer"
)

type user struct {
	name  string
	email string
}

func (c commitObject) Encode() []byte {
	buf := make([]byte, 0, 1024)

	buf = append(buf, "tree"...)
	buf = append(buf, ' ')
	buf = append(buf, c.treeHashHex[:]...)
	buf = append(buf, '\n')

	for _, parentHash := range c.parentHashes {
		buf = append(buf, "parent"...)
		buf = append(buf, ' ')
		buf = append(buf, parentHash[:]...)
		buf = append(buf, '\n')
	}

	buf = append(buf, c.author.encode()...)
	buf = append(buf, c.committer.encode()...)
	buf = append(buf, '\n')
	buf = append(buf, c.message...)
	buf = append(buf, '\n')

	obj := make([]byte, 0, 1024)
	obj = append(obj, "commit"...)
	obj = append(obj, ' ')
	obj = strconv.AppendInt(obj, int64(len(buf)), 10)
	obj = append(obj, 0)
	obj = append(obj, buf...)

	return obj
}

func (s signature) encode() []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf,
		"%s %s <%s> %d %s\n",
		s.role,
		s.name,
		s.email,
		s.seconds,
		s.offset,
	)

	return buf.Bytes()
}
