package main

import (
	"flag"
	"fmt"
	"os"
)

type Command interface {
	Name() string
	Description() string
}

func main() {
	root := newFlagSet("ekko")
	help := root.Bool("h", false, "show help")
	root.Parse(os.Args[1:])

	initCmd := NewInitCmd(newFlagSet("init"))
	hashObjectCmd := NewHashObjectCmd(newFlagSet("hash-object"))
	catFileCmd := NewCatFileCmd(newFlagSet("cat-file"))
	updateIndexCmd := NewUpdateIndexCmd(newFlagSet("update-index"))
	statusCmd := NewStatusCmd(newFlagSet("status"))

	commands := []Command{
		initCmd,
		statusCmd,
		hashObjectCmd,
		catFileCmd,
		updateIndexCmd,
	}

	if *help {
		usage(commands)
		return
	}

	args := root.Args()
	if len(args) == 0 {
		usage(commands)
		os.Exit(1)
	}

	var err error

	subArgs := os.Args[2:]
	switch os.Args[1] {
	case "init":
		err = initCmd.Run(os.Stdout, subArgs...)
	case "hash-object":
		err = hashObjectCmd.Run(os.Stdout, subArgs...)
	case "cat-file":
		err = catFileCmd.Run(os.Stdout, subArgs...)
	case "update-index":
		err = updateIndexCmd.Run(os.Stdout, subArgs...)
	case "status":
		err = statusCmd.Run(os.Stdout, subArgs...)
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

func usage(commands []Command) {
	fmt.Fprintf(os.Stderr, `Ekko - a git-like object system

Usage:
  ekko <command> [options]

Commands:
`)
	for _, cmd := range commands {
		fmt.Fprintf(os.Stderr, "  %-20s%s\n", cmd.Name(), cmd.Description())
	}

	fmt.Fprintf(os.Stderr, `
Global options:
  -h            Show help
`)
}
