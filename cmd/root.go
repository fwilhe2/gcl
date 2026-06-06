package cmd

import (
	"github.com/fwilhe2/gcl/internal/gcl"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	baseDir string
)

var rootCmd = &cobra.Command{
	Use:     "gcl",
	Short:   "git clone wrapper with opinionated directory layout",
	Long:    `git clone wrapper with opinionated directory layout`,
	Args:    cobra.ExactArgs(1),
	Version: version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return gcl.CloneWithOptions(args[0], gcl.CloneOptions{
			BaseDir: baseDir,
		})
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&baseDir, "base-dir", "", "base directory for cloned repositories")
}
