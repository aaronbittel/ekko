package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aaronbittel/ekko/internal/objects"
)

const (
	RED   = "\033[31m"
	GREEN = "\033[32m"
	RESET = "\033[0m"
)

type StatusCmd struct {
	description string

	fs *flag.FlagSet
}

type repoStatus struct {
	Untracked  []string // in working tree only
	Unmodified []string // HEAD == index == working tree

	ModifiedNotStaged []string // working tree differs from index
	ModifiedStaged    []string // index differs from HEAD (ready to commit)

	DeletedNotStaged []string // deleted in working tree, not staged
	DeletedStaged    []string // deletion staged in index

	AddedStaged []string // new files added to index (not in HEAD)
}

func newRepoStatus() *repoStatus {
	return &repoStatus{
		Untracked:         []string{},
		Unmodified:        []string{},
		ModifiedNotStaged: []string{},
	}
}

func NewStatusCmd(fs *flag.FlagSet) *StatusCmd {
	cmd := &StatusCmd{
		description: "Show working tree status",
		fs:          fs,
	}
	cmd.defineFlags()
	return cmd
}

func (cmd *StatusCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-status - Show the working tree status")
		cmd.fs.PrintDefaults()
	}
}

func (cmd *StatusCmd) Description() string {
	return cmd.description
}

func (cmd *StatusCmd) Name() string {
	return cmd.fs.Name()
}

func (cmd *StatusCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	gitRepo, err := findGitRepo()
	if err != nil {
		return err
	}

	var indexEntries []*indexEntry

	indexFile := filepath.Join(gitRepo, ".git", "index")
	indexData, err := os.ReadFile(indexFile)
	if err != nil {
		switch {
		case errors.Is(err, os.ErrNotExist):
			indexEntries = []*indexEntry{}
		default:
			return err
		}
	} else {
		indexEntries, err = readIndex(indexData)
		if err != nil {
			return err
		}
	}

	gitignorePath := filepath.Join(gitRepo, ".gitignore")
	gitignoreFile, err := os.Open(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer gitignoreFile.Close()
	ignoredDirs := readGitignoreFile(gitignoreFile)

	headData, err := os.ReadFile(filepath.Join(gitRepo, ".git", "HEAD"))
	if err != nil {
		return err
	}

	refToCurCommit, found := bytes.CutPrefix(headData, []byte("ref: "))
	if !found {
		return errors.New("invalid index file")
	}
	refToCurCommit = bytes.TrimSpace(refToCurCommit)

	curCommitData, err := os.ReadFile(filepath.Join(gitRepo, ".git", string(refToCurCommit)))
	if err != nil {
		return err
	}
	curCommitHash := string(bytes.TrimRight(curCommitData, "\n"))

	store := objects.NewStore(filepath.Join(gitRepo, ".git", "objects"))

	commitObj, err := store.Open(curCommitHash)
	if err != nil {
		return err
	}
	defer commitObj.Close()

	treeLine, err := commitObj.ReadSlice('\n')
	if err != nil {
		return err
	}
	treeLine = treeLine[:len(treeLine)-1] // remove newline

	treeHashBytes, found := bytes.CutPrefix(treeLine, []byte("tree "))
	if !found || len(treeHashBytes) != 40 {
		return errors.New("invalid commit object")
	}

	treeHash := string(treeHashBytes)
	treeFile, err := os.Open(filepath.Join(gitRepo, ".git", "objects", treeHash[:2], treeHash[2:]))
	if err != nil {
		return err
	}
	defer treeFile.Close()

	treeEntries, err := objects.ParseTreeObject(treeFile)
	if err != nil {
		return err
	}

	repoStatus := newRepoStatus()

	var indexIndex, treeIndex int
	for indexIndex < len(indexEntries) && treeIndex < len(treeEntries) {
		curIndex := indexEntries[indexIndex]
		curObj := treeEntries[treeIndex]

		if curIndex.path == curObj.Name {
			// must be blobs?
			if curIndex.hash == curObj.Hash {
				fmt.Printf("%s == %s\n", curIndex.path, curObj.Name)
			} else {
				fmt.Printf("%s != %s\n", curIndex.path, curObj.Name)
			}
			indexIndex += 1
			treeIndex += 1
			continue
		}

		path := filepath.Join(gitRepo, ".git", "objects", curObj.Hash[:2], curObj.Hash[2:])
		pf, err := os.Open(path)
		if err != nil {
			return err
		}
		// TODO: close earlier in loop
		defer pf.Close()

		obj, err := objects.NewReader(pf)
		if err != nil {
			return err
		}
		// TODO: close earlier in loop
		defer obj.Close()

		if curIndex.path < curObj.Name {
			switch obj.Kind {
			case objects.KindBlob: // do nothing
			case objects.KindTree:
				fmt.Printf("get all entries for %q. Skipping...\n", curObj.Name)
			default:
				return errors.New("illegal kind in tree object")
			}
			indexIndex += 1
		} else {
			switch obj.Kind {
			case objects.KindBlob: // do nothing
			case objects.KindTree:
				fmt.Printf("get all entries for %q. Skipping...\n", curObj.Name)
			default:
				return errors.New("illegal kind in tree object")
			}
			treeIndex += 1
		}
	}

	if err := indexVsWorkingtree(repoStatus, indexEntries, gitRepo, ignoredDirs); err != nil {
		return err
	}

	branchName, found := bytes.CutPrefix(headData, []byte("ref: refs/heads/"))
	if !found {
		return errors.New("invalid HEAD file format")
	}
	branchName = bytes.TrimSpace(branchName)

	fmt.Fprintf(w, "On branch %s\n", branchName)
	fmt.Fprintln(w)

	if len(repoStatus.Unmodified) > 0 {
		fmt.Fprintln(w, "Changes to be committed:")
		for _, path := range repoStatus.Unmodified {
			fmt.Fprintf(w, "\t%s%s%s\n", GREEN, path, RESET)
		}
		fmt.Fprintln(w)
	}

	if len(repoStatus.ModifiedNotStaged) > 0 {
		fmt.Fprintln(w, "Changes not staged for commit:")
		for _, path := range repoStatus.ModifiedNotStaged {
			msg := fmt.Sprintf("%smodified:   %s%s", RED, path, RESET)
			fmt.Fprintf(w, "\t%s\n", msg)
		}
		fmt.Fprintln(w)
	}

	if len(repoStatus.Untracked) > 0 {
		fmt.Fprintln(w, "Untracked files:")
		fmt.Fprintf(w, "\t")
		for _, path := range repoStatus.Untracked {
			msg := fmt.Sprintf("%s%s%s   ", RED, path, RESET)
			fmt.Fprint(w, msg)
		}
		fmt.Fprintln(w)
	} else {
		fmt.Fprintln(w, "no changes added to commit")
	}

	return nil
}

func indexVsWorkingtree(repoStatus *repoStatus, entries []*indexEntry, root string, ignoredDirs []string) error {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, ".git") {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		p, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		for _, dir := range ignoredDirs {
			if strings.HasPrefix(p, dir) {
				return filepath.SkipDir
			}
		}

		for _, entry := range entries {
			if entry.path == p {
				modified, err := isModified(p, entry)
				if err != nil {
					return err
				}
				if modified {
					repoStatus.ModifiedNotStaged = append(repoStatus.ModifiedNotStaged, p)
				} else {
					repoStatus.Unmodified = append(repoStatus.Unmodified, p)
				}
				return nil
			}
		}

		repoStatus.Untracked = append(repoStatus.Untracked, p)

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func isModified(path string, entry *indexEntry) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	ok, err := sameComputeHash(f, entry)
	if err != nil {
		return false, err
	}

	return !ok, nil
}

func sameComputeHash(r io.Reader, entry *indexEntry) (bool, error) {
	object, err := objects.BlobFromReader(r)
	if err != nil {
		return false, err
	}
	defer object.Close()

	hashBytes, err := object.Write(io.Discard)
	if err != nil {
		return false, err
	}

	return bytes.Equal([]byte(entry.hash), hashBytes), nil
}

func readGitignoreFile(r io.Reader) []string {
	lines := []string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}
