package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerializeTree(t *testing.T) {
	sha1 := newSeededSha(t, 1)
	sha2 := newSeededSha(t, 2)
	sha3 := newSeededSha(t, 3)

	tests := []struct {
		name        string
		entries     []treeEntry
		wantHeader  []byte
		wantEntries [][]byte
	}{
		{
			name:       "no entries",
			entries:    []treeEntry{},
			wantHeader: []byte("tree 0\x00"),
		},
		{
			name: "regular file",
			entries: []treeEntry{
				treeEntry{typ: EntryRegularFile, hash: sha1, name: "test.txt"},
			},
			wantHeader: []byte("tree 36\x00"),
			wantEntries: [][]byte{
				append([]byte("100644 test.txt\x00"), sha1[:]...),
			},
		},
		{
			name: "regular file, executable file",
			entries: []treeEntry{
				treeEntry{typ: EntryExecutableFile, hash: sha1, name: "exe.sh"},
				treeEntry{typ: EntryRegularFile, hash: sha2, name: "test.txt"},
			},
			wantHeader: []byte("tree 70\x00"),
			wantEntries: [][]byte{
				append([]byte("100755 exe.sh\x00"), sha1[:]...),
				append([]byte("100644 test.txt\x00"), sha2[:]...),
			},
		},
		{
			name: "files and tree",
			entries: []treeEntry{
				treeEntry{typ: EntryExecutableFile, hash: sha1, name: "exe.sh"},
				treeEntry{typ: EntryTree, hash: sha2, name: "dir"},
				treeEntry{typ: EntryRegularFile, hash: sha3, name: "test.txt"},
			},
			wantHeader: []byte("tree 100\x00"),
			wantEntries: [][]byte{
				append([]byte("100755 exe.sh\x00"), sha1[:]...),
				append([]byte("40000 dir\x00"), sha2[:]...),
				append([]byte("100644 test.txt\x00"), sha3[:]...),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializeTree(tt.entries)
			want := tt.wantHeader
			for _, entry := range tt.wantEntries {
				want = append(want, entry...)
			}
			assert.Equal(t, want, got)
		})
	}
}

// func TestWriteTree(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		treeObj *treeObject
// 		want    gitSha1
// 	}{
// 		{
// 			name:    "no entries",
// 			treeObj: &treeObject{},
// 			want:    newTestSha1(t, "4b825dc642cb6eb9a060e54bf8d69288fbee4904"),
// 		},
// 		{
// 			name: "single regular file",
// 			treeObj: &treeObject{
// 				blobs: []*blobEntry{
// 					&blobEntry{
// 						mode: blobModeRegular, hash: newTestSha1(t, "d670460b4b4aece5915caf5c68d12f560a9fe3e4"), name: "test.txt",
// 					},
// 				},
// 			},
// 			want: newTestSha1(t, "80865964295ae2f11d27383e5f9c0b58a8ef21da"),
// 		},
// 		{
// 			name: "regular, executable file",
// 			treeObj: &treeObject{
// 				blobs: []*blobEntry{
// 					&blobEntry{mode: blobModeRegular, hash: newTestSha1(t, "d670460b4b4aece5915caf5c68d12f560a9fe3e4"), name: "test.txt"},
// 					&blobEntry{mode: blobModeExecutable, hash: newTestSha1(t, "e707a31975a22c47e3645b90f76adf78498b8c0e"), name: "exe.sh"},
// 				},
// 			},
// 			want: newTestSha1(t, "f1aa8c4fe32d514994133f2250c97fb828b4477b"),
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			memStore := MemoryStore{}
// 			got, err := writeTree(tt.treeObj, memStore)
// 			require.NoError(t, err)
// 			for _, content := range memStore {
// 				t.Log("content", content)
// 			}
// 			assert.Equal(t, tt.want, got)
// 		})
// 	}
// }

func TestCreateTree(t *testing.T) {
	var (
		hashA = newTestSha1(t, "1111111111111111111111111111111111111111")
		hashB = newTestSha1(t, "2222222222222222222222222222222222222222")
		hashC = newTestSha1(t, "3333333333333333333333333333333333333333")
		hashD = newTestSha1(t, "4444444444444444444444444444444444444444")
		hashE = newTestSha1(t, "5555555555555555555555555555555555555555")
		hashF = newTestSha1(t, "6666666666666666666666666666666666666666")
		hashG = newTestSha1(t, "7777777777777777777777777777777777777777")
		hashH = newTestSha1(t, "8888888888888888888888888888888888888888")
		hashI = newTestSha1(t, "9999999999999999999999999999999999999999")
		hashJ = newTestSha1(t, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		hashK = newTestSha1(t, "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB")
		hashL = newTestSha1(t, "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC")
		hashM = newTestSha1(t, "DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD")
		hashN = newTestSha1(t, "EEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE")
		hashO = newTestSha1(t, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	)

	tests := []struct {
		name         string
		indexEntries []*indexEntry
		want         *treeObject
	}{
		{
			name:         "empty index",
			indexEntries: []*indexEntry{},
			want:         newTreeRoot(),
		},
		{
			name:         "single file",
			indexEntries: []*indexEntry{&indexEntry{path: "test.txt", gitSha: hashA, mode: RegularFile}},
			want: &treeObject{
				blobs: []*blobEntry{&blobEntry{mode: blobModeRegular, hash: hashA, name: "test.txt"}},
			},
		},
		{
			name: "multiple files in root",
			indexEntries: []*indexEntry{
				&indexEntry{path: "evil.exe", gitSha: hashB, mode: ExecutableFile},
				&indexEntry{path: "test.txt", gitSha: hashA, mode: RegularFile},
			},
			want: &treeObject{
				blobs: []*blobEntry{
					&blobEntry{mode: blobModeExecutable, hash: hashB, name: "evil.exe"},
					&blobEntry{mode: blobModeRegular, hash: hashA, name: "test.txt"},
				},
			},
		},
		{
			name: "flat index with one directory",
			indexEntries: []*indexEntry{
				&indexEntry{path: "evil.exe", gitSha: hashB, mode: ExecutableFile},
				&indexEntry{path: "test.txt", gitSha: hashA, mode: RegularFile},
				&indexEntry{path: "dir/another_file", gitSha: hashC, mode: RegularFile},
			},
			want: &treeObject{
				blobs: []*blobEntry{
					&blobEntry{mode: blobModeExecutable, hash: hashB, name: "evil.exe"},
					&blobEntry{mode: blobModeRegular, hash: hashA, name: "test.txt"},
				},
				trees: []*treeObject{
					&treeObject{
						name: "dir",
						blobs: []*blobEntry{
							&blobEntry{mode: blobModeRegular, hash: hashC, name: "another_file"},
						},
					},
				},
			},
		},
		{
			name: "multiple sibling directories",
			indexEntries: []*indexEntry{
				&indexEntry{path: "bin/run.sh", gitSha: hashA, mode: ExecutableFile},
				&indexEntry{path: "bin/tool", gitSha: hashB, mode: RegularFile},
				&indexEntry{path: "src/main.go", gitSha: hashC, mode: RegularFile},
			},
			want: &treeObject{
				trees: []*treeObject{
					&treeObject{
						name: "bin",
						blobs: []*blobEntry{
							&blobEntry{mode: blobModeExecutable, hash: hashA, name: "run.sh"},
							&blobEntry{mode: blobModeRegular, hash: hashB, name: "tool"},
						},
					},
					&treeObject{
						name: "src",
						blobs: []*blobEntry{
							&blobEntry{mode: blobModeRegular, hash: hashC, name: "main.go"},
						},
					},
				},
			},
		},
		{
			name: "multi-level tree with nested dirs, mixed modes, and sibling directories",
			indexEntries: []*indexEntry{
				&indexEntry{path: "bin/run.sh", gitSha: hashA, mode: ExecutableFile},
				&indexEntry{path: "bin/tool", gitSha: hashB, mode: ExecutableFile},

				&indexEntry{path: "src/main.go", gitSha: hashC, mode: RegularFile},
				&indexEntry{path: "src/lib/util.go", gitSha: hashD, mode: RegularFile},
				&indexEntry{path: "src/lib/internal/helper.go", gitSha: hashE, mode: RegularFile},

				&indexEntry{path: "src/lib/internal/crypto/aes.go", gitSha: hashF, mode: RegularFile},
				&indexEntry{path: "src/lib/internal/crypto/rsa.go", gitSha: hashG, mode: RegularFile},

				&indexEntry{path: "docs/readme.md", gitSha: hashH, mode: RegularFile},
				&indexEntry{path: "docs/specs/tree.md", gitSha: hashI, mode: RegularFile},

				&indexEntry{path: "assets/logo.png", gitSha: hashJ, mode: RegularFile},
				&indexEntry{path: "assets/icons/github.png", gitSha: hashK, mode: RegularFile},
				&indexEntry{path: "assets/icons/gitlab.png", gitSha: hashL, mode: RegularFile},

				&indexEntry{path: "scripts/deploy/prod.sh", gitSha: hashM, mode: ExecutableFile},
				&indexEntry{path: "scripts/deploy/staging.sh", gitSha: hashN, mode: ExecutableFile},
				&indexEntry{path: "scripts/test/unit.sh", gitSha: hashO, mode: ExecutableFile},
			},

			want: &treeObject{
				trees: []*treeObject{
					&treeObject{
						name: "bin",
						blobs: []*blobEntry{
							&blobEntry{mode: blobModeExecutable, hash: hashA, name: "run.sh"},
							&blobEntry{mode: blobModeExecutable, hash: hashB, name: "tool"},
						},
					},
					&treeObject{
						name: "src",
						blobs: []*blobEntry{
							&blobEntry{mode: blobModeRegular, hash: hashC, name: "main.go"},
						},
						trees: []*treeObject{
							&treeObject{
								name: "lib",
								blobs: []*blobEntry{
									&blobEntry{mode: blobModeRegular, hash: hashD, name: "util.go"},
								},
								trees: []*treeObject{
									&treeObject{
										name: "internal",
										blobs: []*blobEntry{
											&blobEntry{mode: blobModeRegular, hash: hashE, name: "helper.go"},
										},
										trees: []*treeObject{
											&treeObject{
												name: "crypto",
												blobs: []*blobEntry{
													&blobEntry{mode: blobModeRegular, hash: hashF, name: "aes.go"},
													&blobEntry{mode: blobModeRegular, hash: hashG, name: "rsa.go"},
												},
											},
										},
									},
								},
							},
						},
					},
					&treeObject{
						name: "docs",
						blobs: []*blobEntry{
							&blobEntry{mode: blobModeRegular, hash: hashH, name: "readme.md"},
						},
						trees: []*treeObject{
							&treeObject{
								name: "specs",
								blobs: []*blobEntry{
									&blobEntry{mode: blobModeRegular, hash: hashI, name: "tree.md"},
								},
							},
						},
					},
					&treeObject{
						name: "assets",
						blobs: []*blobEntry{
							&blobEntry{mode: blobModeRegular, hash: hashJ, name: "logo.png"},
						},
						trees: []*treeObject{
							&treeObject{
								name: "icons",
								blobs: []*blobEntry{
									&blobEntry{mode: blobModeRegular, hash: hashK, name: "github.png"},
									&blobEntry{mode: blobModeRegular, hash: hashL, name: "gitlab.png"},
								},
							},
						},
					},
					&treeObject{
						name: "scripts",
						trees: []*treeObject{
							&treeObject{
								name: "deploy",
								blobs: []*blobEntry{
									&blobEntry{mode: blobModeExecutable, hash: hashM, name: "prod.sh"},
									&blobEntry{mode: blobModeExecutable, hash: hashN, name: "staging.sh"},
								},
							},
							&treeObject{
								name: "test",
								blobs: []*blobEntry{
									&blobEntry{mode: blobModeExecutable, hash: hashO, name: "unit.sh"},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createTree(tt.indexEntries)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncodeBlobEntry(t *testing.T) {
	tests := []struct {
		name  string
		entry *blobEntry
		want  []byte
	}{
		{
			name: "regular file",
			entry: &blobEntry{
				mode: blobModeRegular,
				hash: newTestSha1(t, "1111111111111111111111111111111111111111"),
				name: "test.txt",
			},
			want: []byte{
				0x31, 0x30, 0x30, 0x36, 0x34, 0x34,
				0x20,
				0x74, 0x65, 0x73, 0x74, 0x2E, 0x74, 0x78, 0x74,
				0x00,
				0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11,
				0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11,
			},
		},
		{
			name: "executable file",
			entry: &blobEntry{
				mode: blobModeExecutable,
				hash: newTestSha1(t, "2222222222222222222222222222222222222222"),
				name: "test.txt",
			},
			want: []byte{
				0x31, 0x30, 0x30, 0x37, 0x35, 0x35,
				0x20,
				0x74, 0x65, 0x73, 0x74, 0x2E, 0x74, 0x78, 0x74,
				0x00,
				0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
				0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.encode()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetTree(t *testing.T) {
	tests := []struct {
		name string
		tree *treeObject
		path string
		ret  *treeObject
		want *treeObject
	}{
		{
			name: "create dir",
			tree: newTreeRoot(),
			path: "dir/hello",
			ret:  newTreeObject("dir"),
			want: &treeObject{
				trees: []*treeObject{
					&treeObject{name: "dir"},
				},
			},
		},
		{
			name: "existing dir",
			tree: &treeObject{
				trees: []*treeObject{
					&treeObject{name: "dir"},
				},
			},
			path: "dir/hello",
			ret:  newTreeObject("dir"),
			want: &treeObject{
				trees: []*treeObject{
					&treeObject{name: "dir"},
				},
			},
		},
		{
			name: "multi dir create",
			tree: newTreeRoot(),
			path: "dir/subdir/anotherdir/hello",
			ret:  newTreeObject("anotherdir"),
			want: &treeObject{
				trees: []*treeObject{
					&treeObject{
						name: "dir",
						trees: []*treeObject{
							&treeObject{
								name: "subdir",
								trees: []*treeObject{
									&treeObject{name: "anotherdir"},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tree.getTree(tt.path)
			assert.Equal(t, tt.ret, got)
			assert.Equal(t, tt.want, tt.tree)
		})
	}
}

type MemoryStore map[gitSha1][]byte

func (m MemoryStore) Put(hash gitSha1, data []byte) error {
	if _, ok := m[hash]; !ok {
		m[hash] = data
	} else {
		panic("hash already exists")
	}
	return nil
}
