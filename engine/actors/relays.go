package actors

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

func StartRelaysForPublishing(relays []string) chan nostr.Event {
	sendChan := make(chan nostr.Event)
	var chans []chan nostr.Event
	for _, s := range relays {
		chans = append(chans, make(chan nostr.Event))
		relay, err := nostr.RelayConnect(context.Background(), s)
		if err != nil {
			panic(err)
		}
		go func() {
			select {
			case e := <-chans[len(chans)-1]:
				_, err := relay.Publish(context.Background(), e)
				if err != nil {
					LogCLI(err.Error(), 2)
				}
			}
		}()
	}
	go func() {
		select {
		case e := <-sendChan:
			for _, events := range chans {
				go func(e nostr.Event, events chan nostr.Event) {
					events <- e
				}(e, events)
			}
		}
	}()
	return sendChan
}
