package main

import (
	"os"

	"github.com/joaoajmatos/oz/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
