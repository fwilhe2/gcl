package cmd

import (
	"github.com/fwilhe2/gcl/internal/gcl"

	"github.com/spf13/cobra"
)

var (
	version        = "dev"
	baseDir        string
	depth          int
	skipSubmodules bool
)

var rootCmd = &cobra.Command{
	Use:     "gcl <repository-url>",
	Short:   "git clone wrapper with opinionated directory layout",
	Long:    `git clone wrapper with opinionated directory layout`,
	Args:    cobra.ExactArgs(1),
	Version: version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return gcl.CloneWithOptions(args[0], gcl.CloneOptions{
			BaseDir:        baseDir,
			Depth:          depth,
			SkipSubmodules: skipSubmodules,
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
	rootCmd.Flags().IntVar(&depth, "depth", 0, "create a shallow clone with the given history depth")
	rootCmd.Flags().BoolVar(&skipSubmodules, "no-submodules", false, "skip cloning submodules")
}
