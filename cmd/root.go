package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/fwilhe2/gcl/internal/gcl"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "gcl",
	Short:   "git clone wrapper with opinionated directory layout",
	Long:    `git clone wrapper with opinionated directory layout`,
	Args:    cobra.ExactArgs(1),
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		gitlabToken := os.Getenv("GITLAB_TOKEN")
		githubToken := os.Getenv("GITHUB_TOKEN")
		codebergToken := os.Getenv("CODEBERG_TOKEN")

		if len(args) < 1 {
			os.Exit(0)
		}

		repos, err := gcl.ListRepositories(ctx, args[0], &gcl.Config{
			GitHubToken:   githubToken,
			GitLabToken:   gitlabToken,
			CodebergToken: codebergToken,
		})
		if err != nil {
			log.Fatalf("Error listing repositories: %v", err)
		}

		fmt.Printf("Found %d repositories:\n", len(repos))
		for _, repo := range repos {
			fmt.Printf("  - %s (%s)\n", repo.Name, repo.URL)

			err = gcl.Clone(repo.URL)
			if err != nil {
				log.Fatalln(fmt.Errorf("error %w", err))
			}

		}
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
}
