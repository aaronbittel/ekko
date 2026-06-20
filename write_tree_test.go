package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
				0x62, 0x6C, 0x6F, 0x62,
				0x20,
				0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11,
				0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11,
				0x20,
				0x74, 0x65, 0x73, 0x74, 0x2E, 0x74, 0x78, 0x74,
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
				0x62, 0x6C, 0x6F, 0x62,
				0x20,
				0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
				0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
				0x20,
				0x74, 0x65, 0x73, 0x74, 0x2E, 0x74, 0x78, 0x74,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.Encode()
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
