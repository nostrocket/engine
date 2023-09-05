package snub

import (
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"golang.org/x/exp/slices"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/messaging/relays"
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
	actors.LogCLI(fmt.Sprintf("HEAD commit: %s", head), 4)
	//for _, commit := range repo.CommitEventIDs {
	//	event, err := commit.Event(&repo.Anchor)
	//	if err != nil {
	//		return err
	//	}
	//	fmt.Printf("\n%s\n\n", event)
	//}
	branchName, err := GetCurrentBranch(repo.Anchor.LocalDir)
	if err != nil {
		return err
	}
	branch, ok := repo.Branches[branchName]
	if !ok {
		return fmt.Errorf("could not find branch %s", branchName)
	}
	if branch.Head != head.Hash().String() {
		return fmt.Errorf("HEAD for %s is %s, but current HEAD is %s", branchName, branch.Head, head.Hash().String())
	}
	//GET AND PUBLISH COMMITS
	var include []library.Sha1
	for sha1, sha256 := range branch.CommitGitIDs {
		//this should generate a list of commits for which events do not exist on our relays
		if len(sha256) != 64 {
			include = append(include, sha1)
		}
	}
	err = repo.makeCommitEventsAndPublishToRelays(include)
	if err != nil {
		return err
	}
	for sha1, _ := range branch.CommitGitIDs {
		var success bool = false
		if commit, ok := repo.Commits[sha1]; ok {
			if len(commit.EventID) == 64 && commit.GID == sha1 {
				branch.CommitGitIDs[sha1] = commit.EventID
				branch.CommitEventIDs[commit.EventID] = sha1
				success = true
			}
		}
		if !success {
			return fmt.Errorf("failed to create event for commit %s", sha1)
		}
	}
	repo.Branches[branch.Name] = branch
	var published []string
	for _, sha1 := range include {
		published = append(published, repo.Branches[branch.Name].CommitGitIDs[sha1])
	}
	eventsFromRelay := relays.FetchEvents(repo.Config.GetStringSlice("relays"), nostr.Filters{
		nostr.Filter{
			IDs: published,
		},
	})
	var idsFromRelay []library.Sha256
	if len(eventsFromRelay) != len(published) {
		actors.LogCLI("some events failed to publish", 4)
		var missing []library.Sha256
		for _, event := range eventsFromRelay {
			idsFromRelay = append(idsFromRelay, event.ID)
		}
		for _, s := range published {
			if !slices.Contains(idsFromRelay, s) {
				missing = append(missing, s)
			}
		}

		for _, sha256 := range missing {
			fmt.Println(sha256)
		}

	}

	//GET AND PUBLISH TREES
	commit, err := repo.getTreeFromCommit(branch.Head)
	if err != nil {
		return err
	}
	var objects = make(map[library.Sha1]string)
	err = repo.iterateTree(commit, objects)
	if err != nil {
		return err
	}
	actors.LogCLI(fmt.Sprintf("\nfound %d objects in tree\n", len(objects)), 4)

	//PUBLISH BRANCH
	branch = repo.Branches[branchName]
	branchEvent, err := branch.Event(&repo.Anchor)
	if err != nil {
		return err
	}
	sender := actors.StartRelaysForPublishing(repo.Config.GetStringSlice("relays"))
	sender <- branchEvent
	time.Sleep(time.Second * 5)
	eventsFromRelay = relays.FetchEvents(repo.Config.GetStringSlice("relays"), nostr.Filters{
		nostr.Filter{
			IDs: []string{branchEvent.ID},
		},
	})
	var success bool = false
	for _, event := range eventsFromRelay {
		if event.ID == branchEvent.ID {
			success = true
		}
	}
	if !success {
		actors.LogCLI("failed to publish event", 2)
	}
	actors.LogCLI(fmt.Sprintf("Published repository, subscribe to tag <a %s> to view events", repo.Anchor.childATag()), 4)
	return nil
}

// makeCommitEventsAndPublishToRelays sends all commits in the include list to the relays set in the config file
// it ignores commits if their git identifier is contained in the ignore list, this is useful
// in avoiding republishing commits that already exist as events.
func (r *Repo) makeCommitEventsAndPublishToRelays(include []library.Sha1) error {
	sender := actors.StartRelaysForPublishing(r.Config.GetStringSlice("relays"))
	var sent int64 = 0
	for sha1, commit := range r.Commits {
		if slices.Contains(include, sha1) {
			if len(commit.EventID) != 64 {
				event, err := commit.Event(&r.Anchor)
				if err != nil {
					return err
				}
				sender <- event
				commit.EventID = event.ID
				r.Commits[sha1] = commit
				sent++
			}
		}
	}
	actors.LogCLI(fmt.Sprintf("Sent %d commit events to relays", sent), 4)
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
