package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	root := newFlagSet("ekko")
	root.Usage = usage
	help := root.Bool("h", false, "show help")
	root.Parse(os.Args[1:])

	if *help {
		usage()
		return
	}

	args := root.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	var err error

	subArgs := os.Args[2:]
	switch os.Args[1] {
	case "init":
		err = init_(newFlagSet("init"), os.Stdout, subArgs...)
	case "hash-object":
		err = hashObject(newFlagSet("hash-object"), os.Stdout, subArgs...)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command - %q\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func newFlagSet(name string) *flag.FlagSet {
	return flag.NewFlagSet(name, flag.ExitOnError)
}

func usage() {
	fmt.Fprintf(os.Stderr, `Ekko - a git-like object system

Usage:
  ekko <command> [options]

Commands:
  init          Initialize repository
  hash-object   Hash and optionally store objects

Global options:
  -h            Show help

Examples:
  ekko init
  ekko hash-object -w --stdin
`)
}
