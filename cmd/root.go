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
	Use:   "gcl <repository-url|owner-url>",
	Short: "git clone wrapper with opinionated directory layout",
	Long: `git clone wrapper with opinionated directory layout

Pass a repository URL to clone a single repository, or the URL of an
organization or user (e.g. https://github.com/my-org) to clone all of
its repositories.`,
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
	rootCmd.Version = version
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&baseDir, "base-dir", "", "base directory for cloned repositories")
}
