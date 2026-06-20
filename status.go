package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

	repoStatus, err := status(gitRepo)
	if err != nil {
		return err
	}

	if len(repoStatus.modified) > 0 {
		fmt.Fprintln(w, "modified:")
		for _, path := range repoStatus.modified {
			fmt.Fprintf(w, "  %s\n", path)
		}
	}

	if len(repoStatus.untracked) > 0 {
		fmt.Fprintln(w, "untracked:")
		for _, path := range repoStatus.untracked {
			fmt.Fprintf(w, "  %s\n", path)
		}
	}

	return nil
}

func status(gitRepo string) (*repoStatus, error) {
	indexFile := filepath.Join(gitRepo, ".git", "index")
	data, err := os.ReadFile(indexFile)
	if err != nil {
		return nil, err
	}

	entries, err := readIndex(data)
	if err != nil {
		return nil, err
	}

	repoStatus := newRepoStatus()

	filepath.WalkDir(gitRepo, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, ".git") {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		p, err := filepath.Rel(gitRepo, path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.path == p {
				repoStatus.unmodified = append(repoStatus.unmodified, p)
				return nil
			}
		}

		repoStatus.untracked = append(repoStatus.untracked, p)

		return nil
	})

	return repoStatus, nil
}
