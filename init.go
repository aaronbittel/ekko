package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func init_(w io.Writer, args ...string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)

	if err := fs.Parse(args); err != nil {
		return err
	}

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
			if _, err := os.Create(p); err != nil {
				return err
			}
		}
	}

	headFile := filepath.Join(gitdir, "HEAD")
	if err := os.WriteFile(headFile, []byte("ref: refs/heads/main"), 0664); err != nil {
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
