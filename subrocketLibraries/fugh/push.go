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

func (r *Repo) CreateRepoEvent() (nostr.Event, error) {
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

func (b *Branch) CreateBranchEvent(r *Repo) (nostr.Event, error) {
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

func (r *Repo) FetchAllEvents() (n []nostr.Event, err error) {
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

type Branch struct {
	Name    string           //name of this branch in plaintext, MUST not contain spaces
	Head    library.Sha1     //commit identifier
	ATag    string           //the part of the "a" tag that points to the repo anchor 31228:<pubkey of repo creator>:<repo event d tag>
	DTag    library.Sha256   //random hash
	Commits []library.Sha256 //a list of event IDs for all merged commits for convenience when fetching events
	Length  int64            //the total number of commits in this branch
}

type Repo struct {
	CreatedBy    library.Account
	Name         string
	DTag         library.Sha256    //random hash
	UpstreamDTag library.Sha256    //the upstream repository, only used if this is a fork
	ForkedAt     library.Sha1      //the commit this was forked at
	Maintainers  []library.Account //only used if NOT integrated with nostrocket
	Rocket       library.RocketID  //only used if integrated with nostrocket to get maintainers etc from there
}
