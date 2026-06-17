package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "Usage of Ekko:")
		os.Exit(1)
	}

	var err error

	subArgs := os.Args[2:]
	switch os.Args[1] {
	case "init":
		err = init_(os.Stdout, subArgs...)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command - %q\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
}
