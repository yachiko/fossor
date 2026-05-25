package main

import (
	"os"

	"github.com/yachiko/fossor/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
