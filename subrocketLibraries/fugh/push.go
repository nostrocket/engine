package fugh

import (
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"golang.org/x/exp/slices"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/messaging/relays"
)

func (r *RepoAnchor) GetFullRepoFromEvents() {}

func (r *RepoAnchor) GetAllBlobs(b *Branch) (bm BlobMap, err error) {
	id, err := GetCurrentHeadCommitID(r.LocalDir)
	if err != nil {
		return bm, err
	}
	b.Head = id
	blobMap, err := CreateBlobMap(r.LocalDir)
	if err != nil {
		return nil, err
	}
	for s, bytes := range blobMap {
		fmt.Printf("\n--------------------\n%s\n%s\n\n", s, bytes)
	}

	//get all objects from nostr events (or none if this is new)
	//get all git objects from the local repo
	//create nostr events for all objects that don't exist as events already

	return
}

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
	for _, commit := range b.Commits {
		commits = append(commits, commit)
	}
	branch := nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      31227, //day before torvolds day
		Tags: nostr.Tags{
			nostr.Tag{"name", b.Name},
			nostr.Tag{"d", b.DTag},
			nostr.Tag{"head", b.Head},
			nostr.Tag{"a", b.ATag},
			nostr.Tag(commits),
			nostr.Tag{"len", fmt.Sprintf("%d", len(b.Commits))},
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

func (r *RepoAnchor) FetchAllEvents() (n []nostr.Event, err error) {
	tm := make(nostr.TagMap)
	tm["a"] = []string{"31228:" + r.CreatedBy + ":" + r.DTag}
	n = relays.FetchEvents([]string{"wss://nostr.688.org"}, nostr.Filters{nostr.Filter{
		//Kinds: []int{31228},
		Tags: tm,
		//IDs: []string{repoID},
	}})
	if len(n) == 0 {
		return nil, fmt.Errorf("no events found")
	}
	return
}

type Repo struct {
	Anchor  RepoAnchor
	Commits []Commit
}

type Commit struct {
	//how do we get commits from events and parse them into git objects? hash-object?
	//git cat-file -p df5af114df19730dc1d2936e5819e07273182a76  | git hash-object -t commit --stdin
	GID          library.Sha1
	Author       string //used for legacy operations when pulling from git repos, we don't need author, in snub when merging two or more parents we can see who the authors are from the parents
	Committer    string //the person who publishes the event
	Message      string
	ParentIDs    []library.Sha1
	TreeID       library.Sha1
	EventID      library.Sha256
	LegacyBackup string //the raw text of the commit from existing repo that uses emails as ID and GPG sigs
}

func (c *Commit) parentTag() nostr.Tag {
	var parents = []string{"parents"}
	for _, d := range c.ParentIDs {
		parents = append(parents, d)
	}
	return parents
}

func (c *Commit) Event(ra *RepoAnchor) (commit nostr.Event, err error) {
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
			nostr.Tag{"a", ra.childATag()},
			nostr.Tag{"author", c.Author},
			nostr.Tag{"legacy", c.LegacyBackup},
			c.parentTag(),
		},
		Content: c.Message,
	}
	makeNonce(&commit, c.GID)
	err = commit.Sign(actors.MyWallet().PrivateKey)
	if err != nil {
		return nostr.Event{}, err
	}
	return commit, nil
}

func makeNonce(event *nostr.Event, objectID string) {
	event.Tags = append(event.Tags, nostr.Tag{})
	var nonce int64
	for {
		nonce++
		event.Tags[len(event.Tags)-1] = nostr.Tag{"nonce", fmt.Sprintf("%d", nonce)}
		if event.GetID()[0:4] == objectID[0:4] {
			event.ID = event.GetID()
			return
		}
	}
}

type Tree struct {
	//use mktree instead of hash-object
	Items   []treeObject //the items in the tree
	GID     library.Sha1
	EventID library.Sha256
}

func (t *Tree) Event() (n nostr.Event, err error) {
	return
}

type treeObject struct {
	name     string       //file name
	gID      library.Sha1 //Git ID of the object
	fileMode int64
	blob     Blob //MUST include a Blob XOR Tree
	tree     Tree //MUST include a Blob XOR Tree
}

type Branch struct {
	Name       string           //name of this branch in plaintext, MUST not contain spaces
	Head       library.Sha1     //commit identifier
	ATag       string           //the part of the "a" tag that points to the repo anchor 31228:<pubkey of repo creator>:<repo event d tag>
	DTag       library.Sha256   //random hash
	Commits    []library.Sha256 //a list of event IDs for all merged commits for convenience when fetching events
	Length     int64            //the total number of commits in this branch
	LastUpdate int64            //timestamp of latest update
}

type RepoAnchor struct {
	CreatedBy    library.Account
	Name         string
	DTag         library.Sha256    //random hash
	UpstreamDTag library.Sha256    //the upstream repository, only used if this is a fork
	ForkedAt     library.Sha1      //the commit this was forked at
	Maintainers  []library.Account //only used if NOT integrated with nostrocket
	Rocket       library.RocketID  //only used if integrated with nostrocket to get maintainers etc from there
	LastUpdate   int64             //timestamp of latest update
	LocalDir     string            //the location on the local filesystem if this exists locally
}

func (ra *RepoAnchor) childATag() string {
	return fmt.Sprintf("%d:%s:%s", 31228, ra.CreatedBy, ra.DTag)
}

type Blob struct {
	GID      library.Sha1
	BlobData []byte
	EventID  library.Sha256
}

type BlobMap map[library.Sha1]Blob
