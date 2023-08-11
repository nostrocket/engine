package relays

import (
	"context"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

var tags = make(map[string][]string)
var end = make(chan struct{})
var wait = &deadlock.WaitGroup{}
var lock = &deadlock.Mutex{}

func Subscribe(eventID library.Sha256) {
	lock.Lock()
	defer lock.Unlock()
	close(end)
	wait.Wait()
	end = make(chan struct{})
	sane := library.ValidateSaneExecutionTime()
	defer sane()
	tags["e"] = append(tags["e"], eventID)
	filters := nostr.Filters{
		nostr.Filter{
			Tags: tags,
		}}
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
			ctxsub, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			sub, err := relay.Subscribe(ctxsub, filters)
			if err != nil {
				actors.LogCLI(err.Error(), 1)
				return
			}
			for {
				select {
				case ev := <-sub.Events:
					go func() { sendToConductor <- *ev }()
				case <-end:
					sub.Close()
					relay.Close() //do we need to do this?
					return
				}
			}
		}(url)
	}
}
