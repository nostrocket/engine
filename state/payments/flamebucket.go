package payments

import (
	"time"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/actors"
)

func createRelayAuthEvent(product Product) (n nostr.Event) {
	n.Kind = 15171010
	n.CreatedAt = nostr.Timestamp(time.Now().Unix())
	n.PubKey = actors.MyWallet().Account
	var allowed []string
	allowed = append(allowed, "allow")
	for account, _ := range product.CurrentUsers {
		allowed = append(allowed, account)
	}
	n.Tags = nostr.Tags{allowed}
	n.Content = ""
	n.ID = n.GetID()
	n.Sign(actors.MyWallet().PrivateKey)
	return
}

//events = append(events, nostr.Event{
//	PubKey:    actors.MyWallet().Account,
//	CreatedAt: nostr.Timestamp(time.Now().Unix()),
//	Kind:      15171010,
//	Tags: nostr.Tags{nostr.Tag{"allow", "fe8f4a2e02612e8b5192f76ac52eb641458a56814b4e69374b3c7e125a48a0d2",
//		"c9aed02d02224c064a6694dbb9d464355743b9b330f4f52a43fd05915deb8077",
//		"c47118a6fb01da744990e13c185a41e3aadd140ccb8c441d33989608e8a7cf77"}},
//	Content: "",
//})
