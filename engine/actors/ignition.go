package actors

import (
	"time"

	"github.com/nbd-wtf/go-nostr"
)

func GenerateEvents(publish bool) []nostr.Event {
	return generateEvents(publish)
}

func generateEvents(publish bool) []nostr.Event {
	var stateChangeRequests = nostr.Event{
		PubKey:    MyWallet().Account,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      690000,
		Tags:      nostr.Tags{nostr.Tag{"e", IgnitionEvent, "", "reply"}},
		Content:   "State Change Requests",
	}
	if !publish {
		stateChangeRequests.ID = stateChangeRequests.GetID()
		stateChangeRequests.Sign(MyWallet().PrivateKey)
	} else {
		stateChangeRequests.CreatedAt = nostr.Timestamp(1679905844)
		stateChangeRequests.ID = "2bb52121fb53e3a80ba34596b797cda3203fe2c49b8f1f66e61b0d42f761a9c8"
		stateChangeRequests.Sig = "ea896332472f847eeffd2f5dc2db4e315439d1af0fe6c70e675d7d48653256d859ba7acc39fd08deb5998821e1f3b8a9dcaade0d3d819c29143a369bedc4f27e"
	}

	return []nostr.Event{stateChangeRequests}
}
