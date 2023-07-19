package library

import (
	"strings"

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

func GetFirstReply(e nostr.Event) (string, bool) {
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

func GetAllReplies(e nostr.Event) (r []string) {
	for _, tag := range e.Tags {
		for i, s := range tag {
			if s == "reply" {
				if i == 3 {
					if len(tag[1]) == 64 {
						r = append(r, tag[1])
					}
				}
			}
		}
	}
	return
}

func GetOpData(e nostr.Event, opcode string) (string, bool) {
	for _, tag := range e.Tags {
		if tag.StartsWith([]string{"op"}) {
			if len(tag) > 2 {
				if len(opcode) > 0 {
					ops := strings.Split(tag[1], ".")
					for _, op := range ops {
						if op == opcode {
							return tag[len(tag)-1], true
						}
					}
				} else {
					return tag[len(tag)-1], true
				}
			}

		}
	}
	return "", false
}
