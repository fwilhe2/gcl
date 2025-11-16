package gcl

import (
	"fmt"
	"net/url"
	"os"
	"path"

	"github.com/go-git/go-git/v6"
)

func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // directory does not exist
		}
		return false, err // some other error (e.g., permission)
	}
	return info.IsDir(), nil
}

func Clone(gitUrl string) error {
	urlComponents, err := url.Parse(gitUrl)
	if err != nil {
		return err
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	clonePath := path.Join(homedir, "code", urlComponents.Host, urlComponents.Path)

	exists, err := DirExists(clonePath)
	if err != nil {
		return err
	}
	if exists {
		fmt.Printf("Directory already exists: %s\n", clonePath)
		return nil
	}

	fmt.Printf("Clone Path: %s\n", clonePath)

	err = os.MkdirAll(clonePath, 0o750)
	if err != nil {
		return err
	}

	_, err = git.PlainClone(clonePath, &git.CloneOptions{
		URL:               gitUrl,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          os.Stdout,
	})
	if err != nil {
		return err
	}

	return nil
}
