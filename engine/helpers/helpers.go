package helpers

import (
	"time"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func DeleteEvent(id library.Sha256, reason string) (r nostr.Event) {
	r = nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      5,
		Tags: nostr.Tags{nostr.Tag{
			"e", id},
		},
		Content: reason,
	}
	r.ID = r.GetID()
	r.Sign(actors.MyWallet().PrivateKey)
	return
}
