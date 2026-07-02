package objects

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

type Kind int

const (
	KindBlob Kind = iota
	KindTree
	KindCommit
	KindTag
)

type Object struct {
	Kind         Kind
	ExpectedSize int

	*bufio.Reader
	// source reader
	src io.Closer
	// underlying zlib.Reader. Only stored as io.Closer because all reads should be made
	// through the buffered Reader.
	zr io.Closer
}

func (o *Object) Close() error {
	if o.zr == nil && o.src == nil {
		return nil
	}

	if o.zr == nil && o.src != nil {
		return o.src.Close()
	}

	if o.zr != nil && o.src == nil {
		return o.zr.Close()
	}

	srcCloseErr := o.src.Close()
	zlibCloseErr := o.zr.Close()

	if srcCloseErr != nil {
		return srcCloseErr
	}

	return zlibCloseErr
}

func WriteObject(w io.Writer, kind Kind, data []byte) (hash []byte, err error) {
	zw := zlib.NewWriter(w)
	hw := HashWriter{
		hash: sha1.New(),
		zw:   zw,
	}

	fmt.Fprintf(&hw, "%s %d\x00", kind, len(data))

	if _, err = hw.Write(data); err != nil {
		zw.Close()
		return nil, err
	}

	if closeErr := zw.Close(); closeErr != nil {
		return nil, closeErr
	}

	return hw.hash.Sum(nil), nil
}

func (o *Object) Write(w io.Writer) (hash []byte, err error) {
	zw := zlib.NewWriter(w)
	hw := HashWriter{
		hash: sha1.New(),
		zw:   zw,
	}

	fmt.Fprintf(&hw, "%s %d\x00", o.Kind, o.ExpectedSize)

	_, err = o.WriteTo(&hw)
	if err != nil {
		zw.Close()
		return nil, err
	}

	if closeErr := zw.Close(); closeErr != nil {
		return nil, closeErr
	}

	return hw.hash.Sum(nil), nil
}

type HashWriter struct {
	hash hash.Hash
	zw   *zlib.Writer
}

func (hw *HashWriter) Write(b []byte) (int, error) {
	n, err := hw.zw.Write(b)
	if err != nil {
		return 0, err
	}
	if _, err := hw.hash.Write(b[:n]); err != nil {
		return 0, err
	}
	return n, nil
}

// TODO: instead of accepting a ReadCloser, accept a Reader and try if underlying type
// is a Closer using type assertion. ?
func NewReader(r io.ReadCloser) (*Object, error) {
	zr, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}

	bzr := bufio.NewReader(zr)

	header, err := bzr.ReadSlice(0)
	if err != nil {
		if closeErr := zr.Close(); closeErr != nil {
			return nil, closeErr
		}
		return nil, err
	}
	header = header[:len(header)-1] // skip 0

	kindRaw, sizeRaw, found := bytes.Cut(header, []byte{' '})
	if !found {
		if closeErr := zr.Close(); closeErr != nil {
			return nil, closeErr
		}
		return nil, fmt.Errorf("invalid object header: missing ' '")
	}

	kind, err := kindFromBytes(kindRaw)
	if err != nil {
		if closeErr := zr.Close(); closeErr != nil {
			return nil, closeErr
		}
		return nil, err
	}

	size, err := strconv.Atoi(string(sizeRaw))
	if err != nil {
		if closeErr := zr.Close(); closeErr != nil {
			return nil, closeErr
		}
		return nil, err
	}

	// if this causes performance issues than I can first read from zlib manually and
	// parse the header and then construct a io.MultiReader that reads the remaining
	// bytes from the buf and r, limit it using io.LimitReader and buffering using
	// bufio.Reader.
	bzr = bufio.NewReader(io.LimitReader(bzr, int64(size)))

	return &Object{
		Kind:         kind,
		ExpectedSize: size,

		Reader: bzr,
		src:    r,
		zr:     zr,
	}, nil
}

func BlobFromReader(r io.Reader) (*Object, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return &Object{
		Kind:         KindBlob,
		ExpectedSize: len(data),
		Reader:       bufio.NewReader(bytes.NewBuffer(data)),
	}, nil
}

func (k Kind) String() string {
	switch k {
	case KindBlob:
		return "blob"
	case KindTree:
		return "tree"
	case KindCommit:
		return "commit"
	case KindTag:
		return "tag"
	default:
		panic("invalid kind")
	}
}

func kindFromBytes(b []byte) (Kind, error) {
	switch {
	case bytes.Equal(b, []byte("blob")):
		return KindBlob, nil
	case bytes.Equal(b, []byte("tree")):
		return KindTree, nil
	case bytes.Equal(b, []byte("commit")):
		return KindCommit, nil
	case bytes.Equal(b, []byte("tag")):
		return KindTag, nil
	default:
		return 0, fmt.Errorf("invalid kind %q", b)
	}
}

func ParseKind(kind string) (Kind, error) {
	switch kind {
	case "blob":
		return KindBlob, nil
	case "tree":
		return KindTree, nil
	case "commit":
		return KindCommit, nil
	case "tag":
		return KindTag, nil
	default:
		return 0, fmt.Errorf("invalid kind %q", kind)
	}
}

func GitHashFromObjectPath(path string) (string, error) {
	parts := strings.Split(filepath.ToSlash(path), "/")

	if len(parts) < 2 {
		return "", fmt.Errorf("invalid git object path")
	}

	dir := parts[len(parts)-2]
	file := parts[len(parts)-1]

	if len(dir) != 2 {
		return "", fmt.Errorf("invalid object dir: %s", dir)
	}

	hash := dir + file

	if len(hash) != 40 {
		return "", fmt.Errorf("expected length 40, got %d: %q", len(hash), hash)
	}

	return hash, nil
}
