package fugh

import (
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
)

func GetRepoFromEvent(event nostr.Event) (r Repo, err error) {
	name, ok := library.GetFirstTag(event, "name")
	if !ok {
		return Repo{}, fmt.Errorf("could not find repo name")
	}
	d, ok := library.GetFirstTag(event, "d")
	if !ok {
		return Repo{}, fmt.Errorf("could not find repo d tag")
	}
	maintainers := library.GetTagSlice(event, "maintainers")
	if len(maintainers) == 0 {
		return Repo{}, fmt.Errorf("could not find repo maintainers")
	}
	r.Name = name
	r.Maintainers = maintainers
	r.DTag = d

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
