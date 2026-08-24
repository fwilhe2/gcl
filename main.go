package main

import (
	"fmt"
	"os"

	"github.com/fwilhe2/gcl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
