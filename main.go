package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

type Command interface {
	Name() string
	Description() string
	Run(w io.Writer, args ...string) error
}

func main() {
	root := newFlagSet("ekko")
	help := root.Bool("h", false, "show help")
	root.Parse(os.Args[1:])

	commands := []Command{
		NewInitCmd(newFlagSet("init")),
		NewHashObjectCmd(newFlagSet("hash-object")),
		NewCatFileCmd(newFlagSet("cat-file")),
		NewUpdateIndexCmd(newFlagSet("update-index")),
		NewStatusCmd(newFlagSet("status")),
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

	var (
		cmd     = args[0]
		subArgs = args[1:]
	)

	for _, c := range commands {
		if c.Name() == cmd {
			if err := c.Run(os.Stdout, subArgs...); err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
				os.Exit(1)
			}
			return
		}
	}

	usage(commands)
	fmt.Fprintf(os.Stderr, "error: unknown command - %q\n", os.Args[1])
	os.Exit(1)
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
