package objects

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	// path to .git/objects
	objectsDir string
}

func NewStore(objectsDir string) Store {
	return Store{
		objectsDir: objectsDir,
	}
}

func (s Store) Kind(hash string) (Kind, error) {
	objPath, err := s.GetObjectPath(hash)
	if err != nil {
		return 0, err
	}

	f, err := os.Open(objPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	obj, err := NewReader(f)
	if err != nil {
		return 0, err
	}
	defer obj.Close()

	return obj.Kind, nil
}

func (s Store) ReadTree(hash string) ([]TreeEntry, error) {
	objPath, err := s.GetObjectPath(hash)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(objPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParseTreeObject(f)
}

func (s Store) Open(hash string) (*Object, error) {
	objPath, err := s.GetObjectPath(hash)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(objPath)
	if err != nil {
		return nil, err
	}

	obj, err := NewReader(f)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (s Store) WriteCommit(commit *Commit) (hash string, err error) {
	f, err := os.CreateTemp(".", "hash-object-*")
	if err != nil {
		return "", err
	}
	defer f.Close()

	hashBytes, err := WriteObject(f, KindCommit, commit.Encode())
	if err != nil {
		return "", err
	}

	hash = hex.EncodeToString(hashBytes)

	objPath := filepath.Join(s.objectsDir, hash[:2], hash[2:])
	objDir := filepath.Dir(objPath)

	if err := os.MkdirAll(objDir, 0o755); err != nil {
		return "", err
	}

	if err := os.Rename(f.Name(), objPath); err != nil {
		return "", err
	}

	return hash, nil
}

func (s Store) WriteTree(tree Tree) (hash string, err error) {
	f, err := os.CreateTemp(".", "hash-object-*")
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := tree.Encode()
	if err != nil {
		return "", err
	}
	hashBytes, err := WriteObject(f, KindTree, data)
	if err != nil {
		return "", err
	}

	hash = hex.EncodeToString(hashBytes)

	objPath := filepath.Join(s.objectsDir, hash[:2], hash[2:])
	objDir := filepath.Dir(objPath)

	if err := os.MkdirAll(objDir, 0o755); err != nil {
		return "", err
	}

	if err := os.Rename(f.Name(), objPath); err != nil {
		return "", err
	}

	return hash, nil
}

func (s Store) Write(obj *Object) (hash string, err error) {
	f, err := os.CreateTemp(".", "hash-object-*")
	if err != nil {
		return "", err
	}
	defer f.Close()

	hashBytes, err := obj.Write(f)
	if err != nil {
		return "", err
	}

	hash = hex.EncodeToString(hashBytes)

	objPath := filepath.Join(s.objectsDir, hash[:2], hash[2:])
	objDir := filepath.Dir(objPath)

	if err := os.MkdirAll(objDir, 0o755); err != nil {
		return "", err
	}

	if err := os.Rename(f.Name(), objPath); err != nil {
		return "", err
	}

	return hash, nil
}

// getObjectPath returns the filepath to the object key if it exists.
func (s Store) GetObjectPath(hash string) (string, error) {
	if len(hash) > 40 || len(hash) < 4 {
		return "", fmt.Errorf("not a valid object name %q", hash)
	}

	objectDir := hash[:2]
	objectFile := hash[2:]

	dirEntries, err := os.ReadDir(filepath.Join(s.objectsDir, objectDir))
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
		return "", fmt.Errorf("not a valid object name %s", hash)
	case 1:
		return filepath.Join(s.objectsDir, objectDir, matches[0]), nil
	default:
		return "", &AmbiguousObjectKeyError{
			objectKey:             hash,
			minimalUniquePrefixes: MinimalUniqueObjectPrefix(matches),
		}
	}
}

// TODO: improvent: first check for common prefixes and only search after that
func MinimalUniqueObjectPrefix(names []string) []string {
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
