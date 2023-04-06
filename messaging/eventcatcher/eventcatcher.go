package eventcatcher

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func SubscribeToTree(eChan chan nostr.Event, sendChan chan nostr.Event) {
	relay, err := nostr.RelayConnect(context.Background(), actors.MakeOrGetConfig().GetStringSlice("relaysMust")[0])
	if err != nil {
		panic(err)
	}

	tags := make(map[string][]string)
	tags["e"] = []string{actors.IgnitionEvent}
	var filters nostr.Filters
	filters = []nostr.Filter{{
		//Kinds: []int{1},
		//Authors: []string{pub},
		Tags: tags,
	},
	}

	ctx, cancel := context.WithCancel(context.Background())
	library.LogCLI("Connecting to "+relay.URL, 4)
	sub := relay.Subscribe(ctx, filters)

	go func() {
		for {
			select {
			case e := <-sendChan:
				_, err := relay.Publish(context.Background(), e)
				if err != nil {
					library.LogCLI(err.Error(), 2)
				}
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
			if ev == nil {
				library.LogCLI("Restarting Eventcatcher", 4)
				go SubscribeToTree(eChan, sendChan)
				break L
			} else {
				go func() {
					if ev.Kind >= 640000 && ev.Kind <= 649999 {
						if ok, _ := ev.CheckSignature(); ok {
							eChan <- *ev
						}
					}
				}()
			}
		case <-actors.GetTerminateChan():
			break L
		}
	}
	cancel()
}
