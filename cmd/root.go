package cmd

import (
	"github.com/fwilhe2/gcl/internal/gcl"

	"github.com/spf13/cobra"
)

var (
	baseDir string
	all     bool
)

var rootCmd = &cobra.Command{
	Use:   "gcl <repository-url|owner-url>",
	Short: "git clone wrapper with opinionated directory layout",
	Long: `git clone wrapper with opinionated directory layout

Pass a repository URL to clone a single repository, or the URL of an
organization, user, or group (e.g. https://github.com/my-org) to clone
all of its repositories.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return gcl.CloneWithOptions(args[0], gcl.CloneOptions{
			BaseDir: baseDir,
			All:     all,
		})
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Version = versionInfo()
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.Flags().StringVar(&baseDir, "base-dir", "", "base directory for cloned repositories")
	rootCmd.Flags().BoolVar(&all, "all", false, "clone all repositories of the given owner, even for URLs with multiple path segments (e.g. GitLab subgroups)")
}
