package library

import (
	"github.com/nbd-wtf/go-nostr"
)

func GetFirstTag(e nostr.Event, startsWith string) (string, bool) {
	for _, tag := range e.Tags {
		if tag.StartsWith([]string{startsWith}) {
			return tag.Value(), true
		}
	}
	return "", false
}

func GetReply(e nostr.Event) (string, bool) {
	for _, tag := range e.Tags {
		for i, s := range tag {
			if s == "reply" {
				if i == 3 {
					return tag[1], true
				}
			}
		}
	}
	return "", false
}
