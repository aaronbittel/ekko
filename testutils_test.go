package main

import (
	"flag"
	"io"
)

func newTestFlagSet() *flag.FlagSet {
	fs := new(flag.FlagSet)
	fs.SetOutput(io.Discard)
	return fs
}
