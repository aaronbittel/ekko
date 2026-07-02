package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type InitCmd struct {
	description string

	fs *flag.FlagSet
}

func NewInitCmd(fs *flag.FlagSet) *InitCmd {
	cmd := &InitCmd{
		description: "Initialize repository",
		fs:          fs}

	cmd.defineFlags()
	return cmd
}

func (cmd *InitCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-init - Create an empty Git repository")
		cmd.fs.PrintDefaults()
	}
}

func (cmd *InitCmd) Description() string {
	return cmd.description
}

func (cmd *InitCmd) Name() string {
	return cmd.fs.Name()
}

func (cmd *InitCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	return initRepo(w)
}

func initRepo(w io.Writer) error {
	paths := []string{
		"HEAD",
		"branches/",
		"config",
		"description",
		"hooks/",
		"info/",
		"info/exclude",
		"objects/",
		"objects/info/",
		"objects/pack/",
		"refs/",
		"refs/heads/",
		"refs/tags/",
	}

	gitdir := "./.git"

	if err := os.Mkdir("./.git", 0750); err != nil {
		return err
	}

	for _, path := range paths {
		p := filepath.Join(gitdir, path)
		if strings.HasSuffix(path, "/") {
			if err := os.Mkdir(p, 0750); err != nil {
				return err
			}
		} else {
			if err := os.WriteFile(p, nil, 0664); err != nil {
				return err
			}
		}
	}

	headFile := filepath.Join(gitdir, "HEAD")
	if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0664); err != nil {
		return err
	}

	gitAbspath, err := filepath.Abs(gitdir)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Initialized empty Git repository in %s\n", gitAbspath); err != nil {
		return err
	}

	return nil
}
