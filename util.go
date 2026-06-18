package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	ErrGitRepoNotFound = errors.New("not a git repository (or any of the parent directories): .git")
)

type AmbiguousObjectKeyError struct {
	objectKey             string
	minimalUniquePrefixes []string
}

func (aok *AmbiguousObjectKeyError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "error: short object ID %s is ambiguous\n", aok.objectKey)
	fmt.Fprintln(&sb, "hint:  The candidates are:")
	for _, match := range aok.minimalUniquePrefixes {
		fmt.Fprintf(&sb, "hint:    %s\n", match)
	}
	return sb.String()
}

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
func getObjectPath(gitRepo, objectKey string) (string, error) {
	if len(objectKey) > 40 || len(objectKey) < 4 {
		return "", fmt.Errorf("not a valid object name %q", objectKey)
	}

	objectDir := objectKey[:2]
	objectFile := objectKey[2:]

	objectPath := filepath.Join(gitRepo, "objects", objectDir)
	dirEntries, err := os.ReadDir(objectPath)
	if err != nil {
		return "", err
	}

	matches := []string{}
	for _, dirEntry := range dirEntries {
		if !dirEntry.Type().IsRegular() {
			continue
		}
		if strings.HasPrefix(dirEntry.Name(), objectFile) {
			matches = append(matches, dirEntry.Name())
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("not a valid object name %s", objectKey)
	case 1:
		return filepath.Join(objectPath, matches[0]), nil
	default:
		return "", &AmbiguousObjectKeyError{
			objectKey:             objectKey,
			minimalUniquePrefixes: minimalUniqueObjectPrefix(matches),
		}
	}
}

// TODO: improvent: first check for common prefixes and only search after that
func minimalUniqueObjectPrefix(names []string) []string {
	uniqueObjectPrefixes := make([]string, len(names))

	for j, name := range names {
	outer:
		for i := range 38 {
			for k, other := range names {
				if j == k {
					continue
				}
				if name[i] == other[i] {
					continue outer
				}
			}
			uniqueObjectPrefixes[j] = name[:i+1]
			break
		}
	}

	return uniqueObjectPrefixes
}
