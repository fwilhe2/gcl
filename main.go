package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path"

	"github.com/go-git/go-git/v6"
)

func dirExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: gcl https://example.com/user/repo")
		os.Exit(1)
	}
	gitUrl := os.Args[1]

	urlComponents, err := url.Parse(gitUrl)
	if err != nil {
		log.Fatalln(fmt.Errorf("error %w", err))
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(fmt.Errorf("error %w", err))
	}

	clonePath := path.Join(homedir, "code", urlComponents.Host, urlComponents.Path)

	if dirExists(clonePath) {
		fmt.Printf("Directory already exists: %s\n", clonePath)
		os.Exit(0)
	}

	fmt.Printf("Clone Path: %s\n", clonePath)

	err = os.MkdirAll(clonePath, 0o750)
	if err != nil {
		log.Fatalln(fmt.Errorf("error %w", err))
	}

	_, err = git.PlainClone(clonePath, &git.CloneOptions{
		URL:               gitUrl,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          os.Stdout,
	})
	if err != nil {
		log.Fatalln(fmt.Errorf("error %w", err))
	}
}
