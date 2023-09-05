package snub

import (
	"fmt"
	"strconv"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
)

func GetRepoFromEvent(event nostr.Event) (r RepoAnchor, err error) {
	name, ok := library.GetFirstTag(event, "name")
	if !ok {
		return RepoAnchor{}, fmt.Errorf("could not find repo name")
	}
	d, ok := library.GetFirstTag(event, "d")
	if !ok {
		return RepoAnchor{}, fmt.Errorf("could not find repo d tag")
	}
	maintainers := library.GetTagSlice(event, "maintainers")
	if len(maintainers) == 0 {
		return RepoAnchor{}, fmt.Errorf("could not find repo maintainers")
	}
	r.Name = name
	r.Maintainers = maintainers
	r.DTag = d
	r.CreatedBy = event.PubKey
	r.LastUpdate = event.CreatedAt.Time().Unix()
	forknode, ok := library.GetFirstTag(event, "forknode")
	if ok && len(forknode) == 40 {
		r.ForkedAt = forknode
	}
	rocket, ok := library.GetFirstTag(event, "rocket")
	if ok && len(rocket) == 64 {
		r.Rocket = rocket
	}
	root, ok := library.GetRoot(event)
	if ok && len(root) == 64 {
		r.UpstreamDTag = root
	}
	return
}

func GetBranchFromEvent(event nostr.Event) (r Branch, err error) {
	name, ok := library.GetFirstTag(event, "name")
	if !ok {
		return r, fmt.Errorf("could not find branch name")
	}
	d, ok := library.GetFirstTag(event, "d")
	if !ok {
		return r, fmt.Errorf("could not find branch d tag")
	}
	a, ok := library.GetFirstTag(event, "a")
	if !ok {
		return r, fmt.Errorf("could not find branch d tag")
	}
	r.ATag = nostr.Tag{"a", a}
	r.Name = name
	r.DTag = d
	r.LastUpdate = event.CreatedAt.Time().Unix()
	head, ok := library.GetFirstTag(event, "head")
	if ok && len(head) == 40 {
		r.Head = head
	}
	commits := library.GetTagSlice(event, "commits")
	for _, commit := range commits {
		r.CommitEventIDs[commit] = ""
	}
	length, ok := library.GetFirstTag(event, "len")
	if ok {
		parseInt, err := strconv.ParseInt(length, 10, 64)
		if err == nil {
			r.Length = parseInt
		}
		if err != nil {
			r.Length = int64(len(r.CommitEventIDs))
		}
	}
	return
}
