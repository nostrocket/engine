package relays

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

var cache = make(map[string]nostr.Event)
var cacheMu = &deadlock.Mutex{}

func pushCache(e nostr.Event) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache[e.ID] = e
	publishToBackupRelay(e)
}

func fetchKind0(account library.Account) (e []nostr.Event) {
	for _, event := range cache {
		if event.Kind == 0 && event.PubKey == account {
			e = append(e, event)
		}
	}
	return
}

func FetchCache(id string) (e *nostr.Event, r bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	ev, re := cache[id]
	return &ev, re
}

func publishToBackupRelay(event nostr.Event) {
	if !backupStarted {
		backupStarted = true
		startBackupRelay()
	}
	go func() {
		backupSendChan <- event
	}()
}

var backupStarted = false
var backupSendChan = make(chan nostr.Event)

func startBackupRelay() {
	relay, err := nostr.RelayConnect(context.Background(), "ws://127.0.0.1:45321")
	if err == nil {
		go func() {
			for {
				select {
				case e := <-backupSendChan:
					go func() {
						sane := library.ValidateSaneExecutionTime()
						_, err := relay.Publish(context.Background(), e)
						if err != nil {
							actors.LogCLI(err.Error(), 2)
						}
						sane()
					}()
				}
			}
		}()
	}
	return
}
