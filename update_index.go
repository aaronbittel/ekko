package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// The signature is { 'D', 'I', 'R', 'C' } (stands for "dircache")
var indexSignature = [4]byte{'D', 'I', 'R', 'C'}

type version uint32

var (
	version2 = version(2)
	version3 = version(3) //lint:ignore U1000 currently unused, but here for documentation
	version4 = version(4) //lint:ignore U1000 currently unused, but here for documentation
)

type UpdateIndexCmd struct {
	description string

	updateIndexConfig

	fs *flag.FlagSet
}

type updateIndexConfig struct {
	add   bool
	cinfo *cacheinfo
}

func NewUpdateIndexCmd(fs *flag.FlagSet) *UpdateIndexCmd {
	cmd := &UpdateIndexCmd{
		description: "Add file contents to the index",
		fs:          fs,
	}

	cmd.defineFlags()
	return cmd
}

func (cmd *UpdateIndexCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-update-index - Register file contents in the working tree to the index\n\n")
		cmd.fs.PrintDefaults()
	}

	cmd.fs.BoolVar(&cmd.add, "add", false, "If a specified file isn’t in the index already then it’s added. Default behaviour is to ignore new files.")
	cmd.fs.Func("cacheinfo", "Directly insert the specified info into the index. Expecting <mode>,<object>,<path>.", func(arg string) error {
		parts := strings.SplitN(arg, ",", 3)
		if len(parts) != 3 {
			return errors.New("expect <mode>,<object>,<path>")
		}

		var (
			mode   = parts[0]
			object = parts[1]
			path   = parts[2]
		)

		cmd.cinfo = &cacheinfo{}

		switch mode {
		case "100644":
			cmd.cinfo.mode = RegularFile
		case "100755":
			cmd.cinfo.mode = RegularFile
		case "120000":
			cmd.cinfo.mode = SymbolicLink
		case "160000":
			cmd.cinfo.mode = GitLink
		default:
			return fmt.Errorf("invalid mode %q", parts[0])
		}

		objectSha, err := sha1FromString(object)
		if err != nil {
			return fmt.Errorf("--cacheinfo cannot add %s", object)

		}
		cmd.cinfo.object = objectSha

		cmd.cinfo.path = path

		return nil
	})
}

func (cmd *UpdateIndexCmd) Description() string {
	return cmd.description
}

func (cmd *UpdateIndexCmd) Name() string {
	return cmd.fs.Name()
}

func (cmd *UpdateIndexCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	if !cmd.add {
		cmd.fs.Usage()
		fmt.Fprintln(os.Stderr, "must use '--add'")
	}

	path := cmd.fs.Arg(0)
	if path == "" && cmd.cinfo == nil {
		cmd.fs.Usage()
		fmt.Fprintln(os.Stderr, "no filename provided")
	}

	return runUpdateIndex(path, cmd.cinfo)
}

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
	out = binary.BigEndian.AppendUint32(out, uint32(ih.version))
	out = binary.BigEndian.AppendUint32(out, ih.entryCount)
	return out
}

var ErrInvalidIndexHeader = errors.New("illegal index header format")

func (ih *indexHeader) Decode(b []byte) error {
	if len(b) < 12 {
		return fmt.Errorf("%w: header must be 12 bytes long", ErrInvalidIndexHeader)
	}

	if !bytes.Equal(indexSignature[:], b[:4]) {
		return fmt.Errorf("%w: bad index signature", ErrInvalidIndexHeader)
	}
	copy(ih.signature[:], b[:4])

	ver := version(binary.BigEndian.Uint32(b[4:8]))
	if ver != version2 {
		return fmt.Errorf("%w: expect version 2", ErrInvalidIndexHeader)
	}
	ih.version = ver

	ih.entryCount = binary.BigEndian.Uint32(b[8:12])

	return nil
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
	// always 0 for version 2 => omitted
	extendedFlags uint16

	// Entry path name (variable length) relative to top level directory (without
	// leading slash). '/' is used as path separator. The special path components ".",
	// ".." and ".git" (without quotes) are disallowed. Trailing slash is also
	// disallowed.
	path string
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
	// assume valid: false
	// extended-flag: false (version 2)
	// stage: normal (no conflict)
	entry.flags = entryFlags(min(len(path), 0xFFF))
	entry.extendedFlags = 0
	entry.path = path

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

func (ie *indexEntry) Encode() []byte {
	// 11 * 4 (11 4byte fields, 20 (sha1), +8 max padding)
	out := make([]byte, 0, 11*4+20+len(ie.path)+8)
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
	// extendedFlags always omitted in version 2
	if ie.shouldSetExtendedFlags() {
		out = binary.BigEndian.AppendUint16(out, uint16(ie.extendedFlags))
	}
	out = append(out, []byte(ie.path)...)

	padding := make([]byte, padding(len(out)))
	out = append(out, padding...)

	return out
}

func (ie *indexEntry) Decode(b []byte) (n int, err error) {
	parser := &parser{data: b}

	ie.ctime.seconds = binary.BigEndian.Uint32(parser.readN(4))
	ie.ctime.nanoseconds = binary.BigEndian.Uint32(parser.readN(4))
	ie.mtime.seconds = binary.BigEndian.Uint32(parser.readN(4))
	ie.mtime.nanoseconds = binary.BigEndian.Uint32(parser.readN(4))
	ie.device = binary.BigEndian.Uint32(parser.readN(4))
	ie.inode = binary.BigEndian.Uint32(parser.readN(4))
	ie.mode = gitMode(binary.BigEndian.Uint32(parser.readN(4)))
	ie.userID = binary.BigEndian.Uint32(parser.readN(4))
	ie.groupID = binary.BigEndian.Uint32(parser.readN(4))
	ie.fileSize = binary.BigEndian.Uint32(parser.readN(4))
	copy(ie.gitSha[:], parser.readN(20))
	ie.flags = entryFlags(binary.BigEndian.Uint16(parser.readN(2)))

	filenameLen := ie.filenameLength()
	if filenameLen == 0x0FFF {
		panic("index entry with filename length > 0x0FFF is not supported yet")
	}

	ie.path = string(parser.readN(filenameLen))

	entryLen := 10*4 + 20 + 2 + filenameLen
	expectedPaddingLen := padding(entryLen)

	if !bytes.Equal(make([]byte, expectedPaddingLen), parser.readN(expectedPaddingLen)) {
		return 0, ErrMalformedIndexFile
	}

	return entryLen + expectedPaddingLen, nil
}

func (ie indexEntry) shouldSetExtendedFlags() bool {
	return (ie.flags & 0x4000) != 0
}

func (ie indexEntry) filenameLength() int {
	return int(ie.flags & 0x0FFF)
}

// 1-8 nul bytes as necessary to pad the entry to a multiple of eight bytes
// while keeping the name NUL-terminated.
func padding(size int) int {
	return 8 - (size % 8)
}

type cacheinfo struct {
	mode   gitMode
	object gitSha1
	path   string
}

func runUpdateIndex(path string, cinfo *cacheinfo) error {
	var entry *indexEntry

	if cinfo != nil {
		entry = buildIndexEntryFromCacheinfo(cinfo)
	} else {
		stat, err := getFileStat(path)
		if err != nil {
			return err
		}
		entry, err = buildIndexEntry(path, stat)
		if err != nil {
			return err
		}
	}

	gitRepo, err := findGitRepo()
	if err != nil {
		return err
	}

	indexPath := filepath.Join(gitRepo, ".git", "index")
	var entries []*indexEntry
	_, err = os.Stat(indexPath)
	switch {
	case err == nil:
		// index file exists
		data, err := os.ReadFile(indexPath)
		if err != nil {
			return err
		}
		entries, err = readIndex(data)
		if err != nil {
			return err
		}
	case os.IsNotExist(err):
		entries = []*indexEntry{}
	default:
		return err
	}

	entries = append(entries, entry)

	out := buildIndexFile(entries)
	if err := os.WriteFile(indexPath, out, 0755); err != nil {
		return err
	}

	return nil
}

var ErrMalformedIndexFile = errors.New("malformed index file")

func readIndex(b []byte) ([]*indexEntry, error) {
	body := b[:len(b)-20]
	indexHash := b[len(b)-20:]

	expectedHash := sha1.Sum(body)
	if !bytes.Equal(expectedHash[:], indexHash) {
		return nil, fmt.Errorf("%w: malformed index file hash", ErrMalformedIndexFile)
	}

	parser := &parser{data: b}

	var header indexHeader
	if err := header.Decode(parser.readN(12)); err != nil {
		return nil, err
	}

	indexEntries := make([]*indexEntry, 0, header.entryCount)

	for range header.entryCount {
		var entry indexEntry
		n, err := entry.Decode(parser.current())
		if err != nil {
			return nil, err
		}
		indexEntries = append(indexEntries, &entry)
		parser.off += n
	}

	return indexEntries, nil
}

type parser struct {
	data []byte
	off  int
}

func (p *parser) readN(n int) []byte {
	b := p.data[p.off : p.off+n]
	p.off += n
	return b
}

func (p *parser) current() []byte {
	return p.data[p.off:]
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

func buildIndexEntryFromCacheinfo(cinfo *cacheinfo) *indexEntry {
	var entry indexEntry

	entry.mode = cinfo.mode
	entry.path = cinfo.path
	entry.gitSha = cinfo.object
	entry.flags = entryFlags(min(len(cinfo.path), 0xFFF))

	return &entry
}

func buildIndexFile(entries []*indexEntry) []byte {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].path < entries[j].path
	})

	out := make([]byte, 0, 1024)
	out = append(out, NewIndexHeader(version2, uint32(len(entries))).Encode()...)

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

func sha1FromString(s string) (gitSha1, error) {
	var out gitSha1

	b, err := hex.DecodeString(s)
	if err != nil {
		return out, err
	}

	if len(b) != 20 {
		return out, fmt.Errorf("invalid sha1 length: %d", len(b))
	}

	copy(out[:], b)
	return out, nil
}
