package main

import (
	"os"

	"github.com/ahoma/fossor/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
