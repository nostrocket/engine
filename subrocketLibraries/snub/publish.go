package snub

import (
	"fmt"

	"golang.org/x/exp/slices"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
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
	head, err := repo.Git.Head()
	if err != nil {
		return err
	}
	fmt.Printf("HEAD commit: %s", head)
	//for _, commit := range repo.Commits {
	//	event, err := commit.Event(&repo.Anchor)
	//	if err != nil {
	//		return err
	//	}
	//	fmt.Printf("\n%s\n\n", event)
	//}
	if err := repo.sendCommitsToRelays([]string{}); err != nil {
		return err
	}
	return nil
}

// sendCommitsToRelays sends all commits in the repository to the relays set in the config file
// it ignores commits if their git identifier is contained in the ignore list, this is useful
// in avoiding republishing commits that already exist as events.
func (r *Repo) sendCommitsToRelays(ignore []library.Sha1) error {
	sender := actors.StartRelaysForPublishing(r.Config.GetStringSlice("relays"))
	for sha1, commit := range r.Commits {
		if !slices.Contains(ignore, sha1) {
			event, err := commit.Event(&r.Anchor)
			if err != nil {
				return err
			}
			sender <- event
		}
	}
	//wg := &deadlock.WaitGroup{}
	//for _, s := range r.Config.GetStringSlice("relays") {
	//	actors.LogCLI(fmt.Sprintf("Publishing events to %s", s), 4)
	//	sender := actors.StartRelaysForPublishing([]string{s})
	//	for _, event := range toSend {
	//		wg.Add(1)
	//		//errors := make(chan error)
	//		//success := make(chan bool)
	//		//event.SetExtra("errors", errors)
	//		//event.SetExtra("success", success)
	//		sender <- event
	//
	//		//successCount := 0
	//		//go func() {
	//		//	for {
	//		//		select {
	//		//		case m := <-errors:
	//		//			actors.LogCLI(m, 2)
	//		//			wg.Done()
	//		//		case <-success:
	//		//			successCount++
	//		//			if successCount == len(r.Config.GetStringSlice("relays")) {
	//		//				wg.Done()
	//		//			}
	//		//		}
	//		//	}
	//		//}()
	//	}
	//}
	//
	//wg.Wait()
	return nil
}
