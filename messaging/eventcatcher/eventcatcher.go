package eventcatcher

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

func SubscribeToTree(terminate chan struct{}, eChan chan nostr.Event) {
	relay, err := nostr.RelayConnect(context.Background(), "wss://relay.damus.io")
	if err != nil {
		panic(err)
	}

	tags := make(map[string][]string)
	tags["e"] = []string{"503941a9939a4337d9aef7b92323c353441cb5ebe79f13fed77aeac615116354"}
	var filters nostr.Filters
	filters = []nostr.Filter{{
		//Kinds: []int{1},
		//Authors: []string{pub},
		Tags: tags,
	},
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := relay.Subscribe(ctx, filters)

	go func() {
		<-sub.EndOfStoredEvents
		fmt.Println(36)
		// handle end of stored events (EOSE, see NIP-15)
	}()
	fmt.Println(40)
L:
	for {
		select {
		case ev := <-sub.Events:
			fmt.Println(ev.ID)
		case <-terminate:
			cancel()
			break L
		}
	}
}
