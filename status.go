package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	RED   = "\033[38;2;190;81;81m"
	RESET = "\033[0m"
)

type StatusCmd struct {
	description string

	fs *flag.FlagSet
}

type repoStatus struct {
	untracked  []string
	unmodified []string
	modified   []string
}

func newRepoStatus() *repoStatus {
	return &repoStatus{
		untracked:  []string{},
		unmodified: []string{},
		modified:   []string{},
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

	indexFile := filepath.Join(gitRepo, ".git", "index")
	data, err := os.ReadFile(indexFile)
	if err != nil {
		return err
	}

	entries, err := readIndex(data)
	if err != nil {
		return err
	}

	ignoredDirs := []string{}
	f, err := os.Open(".gitignore")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer f.Close()
	ignoredDirs = readGitignoreFile(f)

	repoStatus, err := status(entries, gitRepo, filepath.WalkDir, ignoredDirs)
	if err != nil {
		return err
	}

	if len(repoStatus.modified) > 0 {
		fmt.Fprintln(w, "Changes not staged for commit:")
		for _, path := range repoStatus.modified {
			msg := fmt.Sprintf("%smodified:   %s%s", RED, path, RESET)
			fmt.Fprintf(w, "\t%s\n", msg)
		}
		fmt.Fprintln(w)
	}

	if len(repoStatus.untracked) > 0 {
		fmt.Fprintln(w, "Untracked files:")
		fmt.Fprintf(w, "\t")
		for _, path := range repoStatus.untracked {
			msg := fmt.Sprintf("%s%s%s   ", RED, path, RESET)
			fmt.Fprint(w, msg)
		}
		fmt.Fprintln(w)
	} else {
		fmt.Fprintln(w, "no changes added to commit")
	}

	return nil
}

type walkFunc func(root string, fn fs.WalkDirFunc) error

func status(entries []*indexEntry, root string, walk walkFunc, ignoredDirs []string) (*repoStatus, error) {
	repoStatus := newRepoStatus()

	walk(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, ".git") {
			return nil
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
				return nil
			}
		}

		for _, entry := range entries {
			if entry.path == p {
				modified, err := isModified(p, entry)
				if err != nil {
					return err
				}
				if modified {
					repoStatus.modified = append(repoStatus.modified, p)
				} else {
					repoStatus.unmodified = append(repoStatus.unmodified, p)
				}
				return nil
			}
		}

		repoStatus.untracked = append(repoStatus.untracked, p)

		return nil
	})

	return repoStatus, nil
}

func isModified(path string, entry *indexEntry) (bool, error) {
	stat, err := getFileStat(path)
	if err != nil {
		return false, err
	}

	if sameModificationTime(stat, entry) {
		return false, nil
	}

	if !sameSize(stat, entry) {
		return true, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	ok, err := sameComputeHash(path, f, entry)
	if err != nil {
		return false, err
	}

	return !ok, nil
}

func sameModificationTime(stat *syscall.Stat_t, entry *indexEntry) bool {
	return stat.Mtim.Sec == int64(entry.mtime.seconds) && stat.Mtim.Nsec == int64(entry.mtime.nanoseconds)
}

func sameComputeHash(path string, r io.Reader, entry *indexEntry) (bool, error) {
	pHash, err := computeObjectHash(path, typeBlob, r)
	if err != nil {
		return false, err
	}
	return bytes.Equal(entry.gitSha[:], pHash[:]), nil
}

func sameSize(stat *syscall.Stat_t, entry *indexEntry) bool {
	return stat.Size == int64(entry.fileSize)
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
