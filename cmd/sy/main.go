package main

import (
	"os"

	"github.com/xtruder/sy/internal/cmd"
)

func main() {
	cmd.SetVersion(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
