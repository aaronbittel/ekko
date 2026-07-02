package main

import (
	"errors"
	"os"
	"path"
	"path/filepath"
)

var (
	ErrGitRepoNotFound = errors.New("not a git repository (or any of the parent directories): .git")
)

// Returns the absolute path to the git repository, searching upwards.
//
// If it is not found, returns [ErrGitRepoNotFound].
func findGitRepo() (string, error) {
	abspath, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	if repo, found := findGitRepoRec(abspath); found {
		return repo, nil
	} else {
		return "", ErrGitRepoNotFound
	}
}

func findGitRepoRec(abspath string) (string, bool) {
	if abspath == "/" {
		if dirExists("/.git/") {
			return "/", true
		}
		return "", false
	}

	gitPath := filepath.Join(abspath, ".git/")
	if dirExists(gitPath) {
		return path.Dir(gitPath), true
	}

	return findGitRepoRec(path.Dir(abspath))
}

func dirExists(dirpath string) bool {
	fi, err := os.Stat(dirpath)
	if err != nil {
		return false
	}

	return fi.IsDir()
}
