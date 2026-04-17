package main

import (
	"os"

	"github.com/oz-tools/oz/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
