package relays

import (
	"context"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
)

func PublishToRelays(events []nostr.Event, relays []string) {
	var wg = &deadlock.WaitGroup{}
	for _, relay := range relays {
		wg.Add(1)
		go func(relay string, events []nostr.Event) {
			mainRelay, err := nostr.RelayConnect(context.Background(), relay)
			if err != nil {
				actors.LogCLI(fmt.Sprintf("could not connect to relay %s: %s", mainRelay, err), 2)
			}
			for _, event := range events {
				_, err := mainRelay.Publish(context.Background(), event)
				if err != nil {
					actors.LogCLI(fmt.Sprintf("could not publish to relay %s: %s", mainRelay, err), 2)
				}
				time.Sleep(time.Second)
			}
			wg.Done()
		}(relay, events)
	}
	wg.Wait()
}
