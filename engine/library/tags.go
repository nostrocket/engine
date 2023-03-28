package library

import (
	"github.com/nbd-wtf/go-nostr"
)

func GetReplayTag(e nostr.Event) (string, bool) {
	for _, tag := range e.Tags {
		if tag.StartsWith([]string{"r"}) {
			return tag.Value(), true
		}
	}
	return "", false
}
