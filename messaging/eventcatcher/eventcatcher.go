package eventcatcher

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
)

const IgnitionEvent string = "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c"

func SubscribeToTree(terminate chan struct{}, eChan chan nostr.Event, sendChan chan nostr.Event) {
	relay, err := nostr.RelayConnect(context.Background(), "wss://nostr.688.org")
	if err != nil {
		panic(err)
	}

	tags := make(map[string][]string)
	tags["e"] = []string{IgnitionEvent}
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
		select {
		case e := <-sendChan:
			_, err := relay.Publish(context.Background(), e)
			if err != nil {
				library.LogCLI(err.Error(), 2)
			}
		}
	}()

	go func() {
		<-sub.EndOfStoredEvents
		// handle end of stored events (EOSE, see NIP-15)
	}()
L:
	for {
		select {
		case ev := <-sub.Events:
			go func() {
				eChan <- *ev
			}()
		case <-terminate:
			break L
		}
	}
	cancel()
}
