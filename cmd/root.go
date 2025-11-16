package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/fwilhe2/gcl/internal/gcl"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:     "gcl",
	Short:   "git clone wrapper with opinionated directory layout",
	Long:    `git clone wrapper with opinionated directory layout`,
	Args:    cobra.ExactArgs(1),
	Version: fmt.Sprintf("%s, %s built %s", Version, Commit, Date),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			os.Exit(0)
		}
		err := gcl.Clone(args[0])
		if err != nil {
			log.Fatalln(fmt.Errorf("error %w", err))
		}
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
}
