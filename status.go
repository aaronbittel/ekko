package main

import (
	"flag"
	"fmt"
	"io"
)

type StatusCmd struct {
	description string

	fs *flag.FlagSet
}

func NewStatusCmd(fs *flag.FlagSet) *StatusCmd {
	cmd := &StatusCmd{
		description: "Show working tree status",
		fs:          fs,
	}
	cmd.defineFlags()
	return cmd
}

func (cmd *StatusCmd) defineFlags() {
	cmd.fs.Usage = func() {
		fmt.Fprintf(cmd.fs.Output(), "ekko-status - Show the working tree status")
		cmd.fs.PrintDefaults()
	}
}

func (cmd *StatusCmd) Description() string {
	return cmd.description
}

func (cmd *StatusCmd) Name() string {
	return cmd.fs.Name()
}

func (cmd *StatusCmd) Run(w io.Writer, args ...string) error {
	if err := cmd.fs.Parse(args); err != nil {
		return err
	}

	return nil
}
