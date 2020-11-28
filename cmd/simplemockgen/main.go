package main

import (
	"os"

	"github.com/theoden9014/simplemock"
)

func main() {
	cmd := simplemock.Command{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	os.Exit(cmd.Run(os.Args[1:]...))
}
