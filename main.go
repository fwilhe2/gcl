package main

import (
	"os"

	"github.com/fwilhe2/gcl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
