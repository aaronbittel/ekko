package main

import (
	"crypto/sha1"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// The signature is { 'D', 'I', 'R', 'C' } (stands for "dircache")
var indexSignature = [4]byte{'D', 'I', 'R', 'C'}

type version [4]byte

var (
	version2 = [4]byte{0, 0, 0, 2}
	version3 = [4]byte{0, 0, 0, 3} //lint:ignore U1000 currently unused, but here for documentation
	version4 = [4]byte{0, 0, 0, 4} //lint:ignore U1000 currently unused, but here for documentation
)

type indexHeader struct {
	signature [4]byte
	// The current supported versions are 2, 3 and 4.
	version version
	// 32-bit number of index entries.
	entryCount uint32
}

func NewIndexHeader(version version, entryCount uint32) *indexHeader {
	return &indexHeader{
		signature:  indexSignature,
		version:    version,
		entryCount: entryCount,
	}
}

func (ih *indexHeader) Encode() []byte {
	out := make([]byte, 0, 12)
	out = append(out, ih.signature[:]...)
	out = append(out, ih.version[:]...)
	out = binary.BigEndian.AppendUint32(out, ih.entryCount)
	return out
}

type indexEntry struct {
	ctime gitTimestamp
	mtime gitTimestamp

	device uint32
	inode  uint32
	mode   gitMode

	userID  uint32
	groupID uint32
	// This is the on-disk size from stat(2), truncated to 32-bit.
	fileSize uint32
	// Object name for the represented object
	gitSha gitSha1

	flags entryFlags
	// assume 0 for simple version 2
	extendedFlags uint16

	// Entry path name (variable length) relative to top level directory (without
	// leading slash). '/' is used as path separator. The special path components ".",
	// ".." and ".git" (without quotes) are disallowed. Trailing slash is also
	// disallowed.
	pathName string
}

type entryFlags uint16

type gitSha1 [20]byte

// gitTimestamp stores seconds and nanoseconds since the Unix epoch.
type gitTimestamp struct {
	seconds     uint32
	nanoseconds uint32
}

type gitMode uint32

const (
	RegularFile    = 0o100644
	ExecutableFile = 0o100755
	SymbolicLink   = 0o120000
	GitLink        = 0o160000
)

type stage uint8

const (
	Normal stage = iota
	Base
	Ours
	Theirs
)

func NewIndexEntry(path string, hash gitSha1, stat *syscall.Stat_t) *indexEntry {
	var entry indexEntry

	entry.ctime.seconds = uint32(stat.Ctim.Sec)
	entry.ctime.nanoseconds = uint32(stat.Ctim.Nsec)
	entry.mtime.seconds = uint32(stat.Mtim.Sec)
	entry.mtime.nanoseconds = uint32(stat.Mtim.Nsec)
	entry.device = uint32(stat.Dev)
	entry.inode = uint32(stat.Ino)

	switch {
	case isRegularFile(stat.Mode):
		if isExecutable(stat.Mode) {
			entry.mode = ExecutableFile
		} else {
			entry.mode = RegularFile
		}
	case isSymbolicLink(stat.Mode):
		entry.mode = SymbolicLink
	default:
		panic("not a valid file mode (regular, symlink)")
	}

	entry.userID = stat.Uid
	entry.groupID = stat.Gid
	entry.fileSize = uint32(stat.Size)
	entry.gitSha = hash

	// TODO: make this configurable
	var flags uint16
	// assume valid: false
	// extended-flag: false (version 2)
	// stage: normal (no conflict)
	nameLen := min(len(path), 0xFFF)
	flags = uint16(nameLen)

	entry.flags = entryFlags(flags)
	entry.extendedFlags = 0
	entry.pathName = path

	return &entry
}

func isExecutable(statMode uint32) bool {
	return (statMode&syscall.S_IXUSR) > 0 &&
		(statMode&syscall.S_IXGRP) > 0 &&
		(statMode&syscall.S_IXOTH) > 0
}

func isRegularFile(statMode uint32) bool {
	return (statMode & syscall.S_IFMT) == syscall.S_IFREG
}

func isSymbolicLink(statMode uint32) bool {
	return (statMode & syscall.S_IFMT) == syscall.S_IFLNK
}

func (ie indexEntry) Encode() []byte {
	// 11 * 4 (11 4byte fields, 20 (sha1), +8 max padding)
	out := make([]byte, 0, 11*4+20+len(ie.pathName)+8)
	out = binary.BigEndian.AppendUint32(out, ie.ctime.seconds)
	out = binary.BigEndian.AppendUint32(out, ie.ctime.nanoseconds)
	out = binary.BigEndian.AppendUint32(out, ie.mtime.seconds)
	out = binary.BigEndian.AppendUint32(out, ie.mtime.nanoseconds)
	out = binary.BigEndian.AppendUint32(out, ie.device)
	out = binary.BigEndian.AppendUint32(out, ie.inode)
	out = binary.BigEndian.AppendUint32(out, uint32(ie.mode))
	out = binary.BigEndian.AppendUint32(out, ie.userID)
	out = binary.BigEndian.AppendUint32(out, ie.groupID)
	out = binary.BigEndian.AppendUint32(out, ie.fileSize)
	out = append(out, ie.gitSha[:]...)
	out = binary.BigEndian.AppendUint16(out, uint16(ie.flags))
	// extendedFlags can be omitted if empty
	if ie.extendedFlags != 0 {
		out = binary.BigEndian.AppendUint16(out, uint16(ie.extendedFlags))
	}
	out = append(out, []byte(ie.pathName)...)

	padding := make([]byte, padding(len(out)))
	out = append(out, padding...)

	return out
}

// 1-8 nul bytes as necessary to pad the entry to a multiple of eight bytes
// while keeping the name NUL-terminated.
func padding(size int) int {
	return 8 - (size % 8)
}

func updateIndex(fs *flag.FlagSet, w io.Writer, args ...string) error {
	fs.Usage = func() {
		fmt.Fprintf(w, "ekko-update-index - Register file contents in the working tree to the index\n\n")
		fs.PrintDefaults()
	}

	var (
		add bool
	)

	fs.BoolVar(&add, "add", false, "If a specified file isn’t in the index already then it’s added. Default behaviour is to ignore new files.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if !add {
		fs.Usage()
		fmt.Fprintln(os.Stderr, "must use '--add'")
	}

	path := fs.Arg(0)
	if path == "" {
		fs.Usage()
		fmt.Fprintln(os.Stderr, "no filename provided")
	}

	return runUpdateIndex(path)
}

func runUpdateIndex(path string) error {
	gitRepo, err := findGitRepo()
	if err != nil {
		return err
	}

	indexPath := filepath.Join(gitRepo, "index")

	stat, err := getFileStat(path)
	if err != nil {
		return err
	}

	entry, err := buildIndexEntry(path, stat)
	if err != nil {
		return err
	}

	out := buildIndexFile([]*indexEntry{entry})
	if err := os.WriteFile(indexPath, out, 0755); err != nil {
		return err
	}

	return nil
}

func buildIndexEntry(path string, stat *syscall.Stat_t) (*indexEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	objSha1, err := computeObjectHash(path, typeBlob, f)
	if err != nil {
		return nil, err
	}

	return NewIndexEntry(path, objSha1, stat), nil
}

func buildIndexFile(entries []*indexEntry) []byte {
	out := make([]byte, 0, 1024)
	out = append(out, NewIndexHeader(version2, 1).Encode()...)

	for _, entry := range entries {
		out = append(out, entry.Encode()...)
	}

	indexSha1 := sha1.Sum(out)
	out = append(out, indexSha1[:]...)

	return out
}

func getFileStat(path string) (*syscall.Stat_t, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("could not get stat_t of file %s", path)
	}

	return stat, nil
}
