package relays

import (
	"context"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

var relays = []string{"ws://127.0.0.1:45321", "wss://nostr.688.org", "wss://nos.lol", "wss://relay.damus.io", "wss://blastr.f7z.xyz", "wss://nostr.mutinywallet.com"}

func FetchLatestProfile(account library.Account) (n nostr.Event, b bool) {
	sane := library.ValidateSaneExecutionTime()
	defer sane()
	events := make(map[string]nostr.Event)
	eventsMu := &deadlock.Mutex{}
	filters := nostr.Filters{
		nostr.Filter{
			Kinds:   []int{0},
			Authors: []string{account},
		}}
	wait := &deadlock.WaitGroup{}
	for _, url := range relays {
		wait.Add(1)
		go func(url string) {
			defer wait.Done()
			ctx := context.Background()
			relay, err := nostr.RelayConnect(ctx, url)
			if err != nil {
				//actors.LogCLI(err.Error(), 1)
				return
			}
			ctxsub, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			sub, err := relay.Subscribe(ctxsub, filters)
			if err != nil {
				actors.LogCLI(err.Error(), 1)
				return
			}
		L:
			for {
				select {
				case ev := <-sub.Events:
					eventsMu.Lock()
					events[ev.ID] = *ev
					pushCache(*ev)
					eventsMu.Unlock()
				case <-time.After(time.Second * 6):
					go func() {
						sub.Close()
						relay.Close()
					}()
					break L
				}
			}

		}(url)
	}
	wait.Wait()
	var timestamp nostr.Timestamp
	for _, event := range events {
		if event.CreatedAt > timestamp {
			go func() { sendToConductor <- n }()
			b = true
			n = event
			timestamp = event.CreatedAt
		}
	}
	if !b {
		actors.LogCLI("could not find profile for account", 1)
	}
	return
}
