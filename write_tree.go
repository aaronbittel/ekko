package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aaronbittel/ekko/internal/objects"
)

type WriteTreeCmd struct {
	description string

	fs *flag.FlagSet
}

func NewWriteTreeCmd(fs *flag.FlagSet) *WriteTreeCmd {
	cmd := &WriteTreeCmd{
		description: "Create a tree object",
		fs:          fs,
	}
	cmd.defineFlags()
	return cmd
}

func (cmd *WriteTreeCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-write-tree - Create a tree object from the current index")
		cmd.fs.PrintDefaults()
	}
}

func (cmd *WriteTreeCmd) Description() string {
	return cmd.description
}

func (cmd *WriteTreeCmd) Name() string {
	return cmd.fs.Name()
}

func (cmd *WriteTreeCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	gitRepo, err := findGitRepo()
	if err != nil {
		return err
	}

	store := objects.NewStore(filepath.Join(gitRepo, ".git", "objects"))
	_ = store

	// TODO: handle index file not existing, git creates an empty tree object and writes
	// that to index
	indexFile := filepath.Join(gitRepo, ".git", "index")
	data, err := os.ReadFile(indexFile)
	if err != nil {
		return err
	}

	entries, err := readIndex(data)
	if err != nil {
		return err
	}

	hash, err := writeTree(store, createDirectory(entries))
	if err != nil {
		return err
	}

	fmt.Fprintln(w, hash)

	return nil
}

func writeTree(store objects.Store, dir *directory) (hash string, err error) {
	tree := objects.Tree{}

	var fileIdx, dirIdx int

	for fileIdx < len(dir.files) || dirIdx < len(dir.dirs) {
		if fileIdx >= len(dir.files) {
			for ; dirIdx < len(dir.dirs); dirIdx++ {
				hash, err := writeTree(store, dir.dirs[dirIdx])
				if err != nil {
					return "", err
				}
				tree.Entries = append(tree.Entries, objects.TreeEntry{
					Hash: hash,
					Name: dir.dirs[dirIdx].name,
					Mode: objects.ModeTree,
					Kind: objects.KindTree,
				})
			}
			break
		}

		if dirIdx >= len(dir.dirs) {
			for ; fileIdx < len(dir.files); fileIdx++ {
				tree.Entries = append(tree.Entries, objects.TreeEntry{
					Hash: dir.files[fileIdx].hash,
					Name: dir.files[fileIdx].name,
					Mode: dir.files[fileIdx].mode,
					Kind: objects.KindBlob,
				})
			}
			break
		}

		if dir.dirs[dirIdx].name < dir.files[fileIdx].name {
			hash, err := writeTree(store, dir.dirs[dirIdx])
			if err != nil {
				return "", err
			}
			tree.Entries = append(tree.Entries, objects.TreeEntry{
				Hash: hash,
				Name: dir.dirs[dirIdx].name,
				Mode: objects.ModeTree,
				Kind: objects.KindTree,
			})
			dirIdx++
		} else {
			tree.Entries = append(tree.Entries, objects.TreeEntry{
				Hash: dir.files[fileIdx].hash,
				Name: dir.files[fileIdx].name,
				Mode: dir.files[fileIdx].mode,
				Kind: objects.KindBlob,
			})
			fileIdx++
		}
	}

	return store.WriteTree(tree)
}

type directory struct {
	name  string
	dirs  []*directory
	files []*file
}

type file struct {
	name string
	mode string
	hash string
}

func createDirectory(entries []*indexEntry) *directory {
	directory := new(directory)

	for _, entry := range entries {
		t := directory.getTree(entry.path)
		name := filepath.Base(entry.path)
		t.addBlob(entry.mode, entry.hash, name)
	}

	return directory
}

func (d *directory) getTree(path string) *directory {
	current := d

	parts := strings.Split(path, "/")
outer:
	for _, di := range parts[:len(parts)-1] {
		for _, ts := range current.dirs {
			if ts.name == di {
				current = ts
				continue outer
			}
		}
		current = current.addDir(di)
	}

	return current
}
func (d *directory) addDir(name string) *directory {
	next := &directory{name: name}
	d.dirs = append(d.dirs, next)
	return next
}

func (d *directory) addBlob(mode gitMode, hash string, path string) {
	var f file
	switch mode {
	case RegularFile:
		f.mode = objects.ModeRegularFile
	case ExecutableFile:
		f.mode = objects.ModeExecutableFile
	default:
		panic(fmt.Sprintf("illegal blob mode %v", mode))
	}

	f.hash = hash
	f.name = path

	d.files = append(d.files, &f)
}
