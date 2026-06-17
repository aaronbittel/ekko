package main

import (
	"errors"
	"fmt"
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
			return "/.git/", true
		}
		return "", false
	}

	gitPath := filepath.Join(abspath, ".git/")
	if dirExists(gitPath) {
		return gitPath, true
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

// Searches for the Git repository and, if found, verifies that the
// specified object exists. It returns the corresponding directory and file name.
// func splitObjectKey(objectKey string) (dir, file string, err error) {
// 	if len(objectKey) > 40 || len(objectKey) < 4 {
// 		return "", "", fmt.Errorf("not a valid object name %q", objectKey)
// 	}
//
// 	gitRepo, err := findGitRepo()
// 	if err != nil {
// 		return err
// 	}
//
// 	objectPath := filepath.Join(gitRepo, "")
//
// 	return objectKey[:2], objectKey[2:], nil
// }

// Searches for the Git repository and, if found, verifies that the
// specified object exists. It returns the filepath to the object key.
func getObjectPath(objectKey string) (string, error) {
	if len(objectKey) > 40 || len(objectKey) < 4 {
		return "", fmt.Errorf("not a valid object name %q", objectKey)
	}

	gitRepo, err := findGitRepo()
	if err != nil {
		return "", err
	}

	objectDir := objectKey[:2]
	objectFile := objectKey[2:]

	objectPath := filepath.Join(gitRepo, "objects", objectDir, objectFile)
	if _, err := os.Stat(objectPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("Not a valid object name %s", objectKey)
		} else {
			return "", err
		}
	}

	return objectPath, nil
}
