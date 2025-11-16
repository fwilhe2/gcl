package cmd

import (
	"os"

	"github.com/fwilhe2/gcl/internal/gcl"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gcl",
	Short: "git clone wrapper with opinionated directory layout",
	Long:  `git clone wrapper with opinionated directory layout`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			os.Exit(0)
		}
		gcl.Clone(args[0])
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
}
