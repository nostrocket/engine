package snub

import (
	"fmt"
)

func Publish(options NewRepoOptions) error {
	LoadRocketConfig()
	if len(options.Path) == 0 {
		options.Path = GetCurrentDirectory()
	}
	repo, err := BuildFromExistingRepo(options)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	head, err := repo.Git.Head()
	if err != nil {
		return err
	}
	fmt.Printf("%s", head)
	return nil
}
