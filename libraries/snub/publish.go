package snub

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/nbd-wtf/go-nostr"
	"golang.org/x/exp/slices"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/messaging/relays"
)

func Publish(options NewRepoOptions) error {
	actors.LogCLI("Loading configuration", 4)
	LoadRocketConfig()
	if len(options.Path) == 0 {
		options.Path = GetCurrentDirectory()
	}
	actors.LogCLI("Building a representation of local repository", 4)
	repo, err := BuildFromExistingRepo(options)
	if err != nil {
		return err
	}
	actors.LogCLI("Starting Relays", 4)
	repo.Sender = actors.StartRelaysForPublishing(repo.Config.GetStringSlice("relays"))
	head, err := repo.Git.Head()
	if err != nil {
		return err
	}
	actors.LogCLI(fmt.Sprintf("HEAD commit found: %s", head), 4)
	//for _, commit := range repo.CommitEventIDs {
	//	event, err := commit.Event(&repo.Anchor)
	//	if err != nil {
	//		return err
	//	}
	//	fmt.Printf("\n%s\n\n", event)
	//}
	actors.LogCLI("Building local branch objects", 4)
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
	actors.LogCLI("Building local commit objects", 4)
	var include []library.Sha1
	for sha1, sha256 := range branch.CommitGitIDs {
		//this should generate a list of commits for which events do not exist on our relays
		if len(sha256) != 64 {
			include = append(include, sha1)
		}
	}
	actors.LogCLI("Publishing commit objects", 4)
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
	actors.LogCLI("Checking that the objects were all published", 4)
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
	var trees []library.Sha1
	var blobs []library.Sha1
	for sha1, s := range objects {
		if s == "tree" {
			trees = append(trees, sha1)
		}
		if s == "blob" {
			blobs = append(blobs, sha1)
		}
	}
	actors.LogCLI("Building local tree objects and sending to relays", 4)
	err = repo.makeTreeEventsAndPublishToRelays(trees, blobs)
	if err != nil {
		return err
	}
	actors.LogCLI("Building local blob objects and sending to relays", 4)
	err = repo.makeBlobEventsAndPublishToRelays(blobs)
	if err != nil {
		return err
	}

	//PUBLISH BRANCH
	branch = repo.Branches[branchName]
	branchEvent, err := branch.Event(&repo.Anchor)
	if err != nil {
		return err
	}
	repo.Sender <- branchEvent
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
	actors.LogCLI("Publishing repo anchor event", 4)
	event, err := repo.Anchor.Event()
	if err != nil {
		return err
	}
	repo.Sender <- event
	time.Sleep(time.Second * 2)
	//todo validate all events are on the relay and resend anything that is missing
	actors.LogCLI(fmt.Sprintf("Published repository, subscribe to tag %s to view all events, or to view an example of each event kind, execute the command snub example -a \"%s\"", repo.Anchor.childATag(), repo.Anchor.childATag()[1:]), 4)
	return nil
}

func (r *Repo) makeBlobEventsAndPublishToRelays(blobs []library.Sha1) error {
	for _, sha1 := range blobs {
		blobBytes, err := r.getBinaryBlob(sha1)
		if err != nil {
			return err
		}
		object, err := getGitHashForObject(fmt.Sprintf("%s", blobBytes), "blob")
		if err != nil {
			return err
		}
		if sha1 != object {
			return fmt.Errorf("hash mismatch")
		}
		blob := Blob{
			GID:      sha1,
			BlobData: blobBytes,
		}
		event, err := blob.Event(r)
		if err != nil {
			return err
		}
		r.Sender <- event
		blob.EventID = event.ID
		if len(r.Blobs) == 0 {
			r.Blobs = make(map[library.Sha1]Blob)
		}
		r.Blobs[sha1] = blob
	}
	return nil
}

func (r *Repo) makeTreeEventsAndPublishToRelays(trees []library.Sha1, blobs []library.Sha1) error {
	var sent int64 = 0
	for _, sha1 := range trees {
		object, err := r.Git.TreeObject(plumbing.NewHash(sha1))
		if err != nil {
			return err
		}
		tree := Tree{
			Items: []TreeItem{},
			GID:   sha1,
		}
		for _, entry := range object.Entries {
			var t string
			if slices.Contains(trees, entry.Hash.String()) {
				t = "tree"
			} else {
				if slices.Contains(blobs, entry.Hash.String()) {
					t = "blob"
				}
			}
			if len(t) == 0 {
				panic("this should not happen")
			}
			tree.Items = append(tree.Items, TreeItem{
				Filemode: entry.Mode,
				Name:     entry.Name,
				Hash:     entry.Hash.String(),
				Type:     t,
			})
		}
		err = tree.writeAndValidate()
		if err != nil {
			return err
		}
		event, err := tree.Event(r)
		if err != nil {
			return err
		}
		r.Sender <- event
		sent++
		tree.EventID = event.ID
		if len(r.Trees) == 0 {
			r.Trees = make(map[library.Sha1]Tree)
		}
		r.Trees[tree.GID] = tree
	}
	return nil
}

// makeCommitEventsAndPublishToRelays sends all commits in the include list to the relays set in the config file
// it ignores commits if their git identifier is contained in the ignore list, this is useful
// in avoiding republishing commits that already exist as events.
func (r *Repo) makeCommitEventsAndPublishToRelays(include []library.Sha1) error {
	var sent int64 = 0
	for sha1, commit := range r.Commits {
		if slices.Contains(include, sha1) {
			if len(commit.EventID) != 64 {
				event, err := commit.Event(r)
				if err != nil {
					return err
				}
				r.Sender <- event
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
