package snub

import (
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"golang.org/x/exp/slices"
	"nostrocket/engine/actors"
	"nostrocket/messaging/relays"
)

func (r *RepoAnchor) GetFullRepoFromEvents() {}

//func (r *RepoAnchor) GetAllBlobs(b *Branch) (bm BlobMap, err error) {
//	id, err := GetCurrentHeadCommitID(r.LocalDir)
//	if err != nil {
//		return bm, err
//	}
//	b.Head = id
//	blobMap, err := CreateBlobMap(r.LocalDir)
//	if err != nil {
//		return nil, err
//	}
//	for s, bytes := range blobMap {
//		fmt.Printf("\n--------------------\n%s\n%s\n\n", s, bytes)
//	}
//
//	//get all objects from nostr events (or none if this is new)
//	//get all git objects from the local repo
//	//create nostr events for all objects that don't exist as events already
//
//	return
//}

func (r *RepoAnchor) Event() (nostr.Event, error) {
	//todo validate name doesn't have illegal characters (space etc)
	var maintainers = []string{"maintainers"}
	for _, maintainer := range r.Maintainers {
		maintainers = append(maintainers, maintainer)
	}
	anchor := nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      31228, //torvolds day
		Tags: nostr.Tags{
			nostr.Tag{"rocket", r.Rocket},
			nostr.Tag{"name", r.Name},
			nostr.Tag{"d", r.DTag},
			nostr.Tag{"a", r.UpstreamDTag},
			nostr.Tag{"forknode", r.ForkedAt},
			nostr.Tag(maintainers),
		},
		Content: "",
	}
	anchor.ID = anchor.GetID()
	err := anchor.Sign(actors.MyWallet().PrivateKey)
	if err != nil {
		return nostr.Event{}, err
	}
	return anchor, nil
}

func (b *Branch) Event(r *RepoAnchor) (nostr.Event, error) {
	//todo make the branch locally and get all this data from there
	//todo validate name doesn't have illegal characters (space etc)
	if !slices.Contains(r.Maintainers, actors.MyWallet().Account) {
		return nostr.Event{}, fmt.Errorf("you need to be a maintainer on this repo to create a branch")
	}
	var commits []string
	commits = append(commits, "commits")
	for sha256, _ := range b.CommitEventIDs {
		commits = append(commits, sha256)
	}
	branch := nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      31227, //day before torvolds day
		Tags: nostr.Tags{
			nostr.Tag{"name", b.Name},
			nostr.Tag{"d", b.DTag},
			nostr.Tag{"head", b.Head},
			b.ATag,
			nostr.Tag(commits),
			nostr.Tag{"len", fmt.Sprintf("%d", len(b.CommitEventIDs))},
		},
		Content: "",
	}
	branch.ID = branch.GetID()
	err := branch.Sign(actors.MyWallet().PrivateKey)
	if err != nil {
		return nostr.Event{}, err
	}
	return branch, nil
}

func (r *Repo) FetchAllEvents() (n []nostr.Event, err error) {
	tm := make(nostr.TagMap)
	tm["a"] = []string{"31228:" + r.Anchor.CreatedBy + ":" + r.Anchor.DTag}
	n = relays.FetchEvents(r.Config.GetStringSlice("relays"), nostr.Filters{nostr.Filter{
		//Kinds: []int{31228},
		Tags: tm,
		//IDs: []string{repoID},
	}})
	if len(n) == 0 {
		return nil, fmt.Errorf("no events found")
	}
	return
}

func (c *Commit) parentTag() nostr.Tag {
	var parents = []string{"parents"}
	for _, d := range c.ParentIDs {
		parents = append(parents, d)
	}
	return parents
}

func (c *Commit) Event(r *Repo) (commit nostr.Event, err error) {
	//when migrating from existing git repo, we need to include the full text of the commit to maintain backwards
	//compatibility with legacy repositories that use emails as ID instead of pubkeys, if we don't include this it
	//might become impossible to reproduce the same identifier
	commit = nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      3121,
		Tags: nostr.Tags{
			nostr.Tag{"gid", c.GID},
			nostr.Tag{"tree", c.TreeID},
			r.Anchor.childATag(),
			c.Author.tag(),
			c.Committer.tag(),
			//nostr.Tag{"legacy", c.LegacyBackup}, //I think we might not need this, hashes seem to always work
			c.parentTag(),
		},
		Content: c.Message,
	}
	if len(c.LegacyBackup) > 0 {
		commit.Tags = append(commit.Tags, nostr.Tag{"legacy", c.LegacyBackup})
	}
	makeNonce(&commit, c.GID, r.Config.GetInt64("PoW"))
	err = commit.Sign(actors.MyWallet().PrivateKey)
	if err != nil {
		return nostr.Event{}, err
	}
	return commit, nil
}

func (t *Tree) Event(r *Repo) (n nostr.Event, err error) {
	n = nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      3122,
		Tags: nostr.Tags{
			nostr.Tag{"gid", t.GID},
			r.Anchor.childATag(),
		},
		Content: "snub tree object",
	}
	blobTag := nostr.Tag{"blobs"}
	treeTag := nostr.Tag{"trees"}
	for _, item := range t.Items {
		if item.Type == "blob" {
			blobTag = append(blobTag, fmt.Sprintf("%s:%s:%s", item.Hash, item.Name, item.filemode()))
		}
		if item.Type == "tree" {
			treeTag = append(treeTag, fmt.Sprintf("%s:%s:%s", item.Hash, item.Name, item.filemode()))
		}
	}
	n.Tags = append(n.Tags, blobTag)
	n.Tags = append(n.Tags, treeTag)

	makeNonce(&n, t.GID, r.Config.GetInt64("PoW"))
	err = n.Sign(actors.MyWallet().PrivateKey)
	if err != nil {
		return nostr.Event{}, err
	}
	return
}

func (b *Blob) Event(r *Repo) (n nostr.Event, err error) {
	compressed, err := compressBytes(b.BlobData)
	if err != nil {
		return n, err
	}
	n = nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      3123,
		Tags: nostr.Tags{
			nostr.Tag{"gid", b.GID},
			nostr.Tag{"data", fmt.Sprintf("%x", compressed)},
			r.Anchor.childATag(),
		},
		Content: "snub blob object",
	}
	makeNonce(&n, b.GID, r.Config.GetInt64("PoW"))
	err = n.Sign(actors.MyWallet().PrivateKey)
	if err != nil {
		return nostr.Event{}, err
	}
	return
}

func makeNonce(event *nostr.Event, objectID string, pow int64) {
	//todo make this time bounded and get the best hash possible
	actors.LogCLI(fmt.Sprintf("mining event ID to match git object identifier: %s", objectID[0:4]), 4)
	event.Tags = append(event.Tags, nostr.Tag{})
	var nonce int64
	for {
		nonce++
		event.Tags[len(event.Tags)-1] = nostr.Tag{"nonce", fmt.Sprintf("%d", nonce)}
		if event.GetID()[0:pow] == objectID[0:pow] {
			event.ID = event.GetID()
			return
		}
	}
}

func (ra *RepoAnchor) childATag() (t nostr.Tag) {
	t = append(t, "a")
	t = append(t, fmt.Sprintf("%d:%s:%s", 31228, ra.CreatedBy, ra.DTag))
	return
}
