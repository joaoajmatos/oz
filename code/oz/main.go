package main

import (
	"errors"
	"os"

	"github.com/joaoajmatos/oz/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		exitCode := 1
		var withCode interface{ ExitCode() int }
		if errors.As(err, &withCode) {
			exitCode = withCode.ExitCode()
		}
		os.Exit(exitCode)
	}
}
