package eventcatcher

import (
	"context"
	"fmt"
	"time"

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

func FetchLatestKind0(accounts []library.Account) (nostr.Event, bool) {
	eChan := make(chan nostr.Event)
	fetchKind0(activeRelayConnection, accounts, eChan)
	var latest nostr.Event
	for {
		select {
		case <-time.After(time.Second * 10):
			return nostr.Event{}, false
		case e := <-eChan:
			if e.CreatedAt.After(latest.CreatedAt) {
				latest = e
			}
			if e.Kind == 21 {
				if len(latest.PubKey) == 64 {
					return latest, true
				}
			}
		}
	}
}

func fetchKind0(relay *nostr.Relay, accounts []library.Account, eChan chan nostr.Event) {
	var filters = []nostr.Filter{{
		Kinds: []int{0},
	}}
	if len(accounts) > 0 {
		filters = []nostr.Filter{{
			Kinds:   []int{0},
			Authors: accounts,
		}}
		//fmt.Printf("%#v", filters)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := relay.Subscribe(ctx, filters)
	go func() {
		attempts := 0
		events := make(map[string]nostr.Event)
		for {
			select {
			case ev := <-sub.Events:
				events[ev.ID] = *ev
				pushCache(*ev)
			case <-time.After(time.Second * 2):
				attempts++
				if len(events) > 0 {
					cancel()
					for _, event := range events {
						eChan <- event
					}
					eChan <- nostr.Event{
						Kind: 21,
					}
					return
				}
				if attempts > 5 {
					cancel()
					return
				}
			}
		}
	}()
}

var activeRelayConnection *nostr.Relay
var activeSendChan chan nostr.Event

func PublishEvent(event nostr.Event) {
	if ok, err := event.CheckSignature(); ok && err != nil {
		go func() {
			activeSendChan <- event
		}()
	}
}

func SubscribeToTree(eChan chan nostr.Event, sendChan chan nostr.Event, eose chan bool) {
	activeSendChan = sendChan
	var sleepChan = make(chan bool)
	sleeper(sleepChan)
	relay, err := nostr.RelayConnect(context.Background(), actors.MakeOrGetConfig().GetStringSlice("relaysMust")[0])
	if err != nil {
		actors.LogCLI(fmt.Sprintf("could not connect to relay: %s", err), 0)
	}
	activeRelayConnection = relay
	tags := make(map[string][]string)
	tags["e"] = []string{actors.IgnitionEvent}
	var filters nostr.Filters
	filters = []nostr.Filter{{
		//Kinds: []int{1},
		//Authors: []string{pub},
		Tags: tags,
	}}

	ctx, cancel := context.WithCancel(context.Background())
	actors.LogCLI("Connecting to "+relay.URL, 4)
	sub := relay.Subscribe(ctx, filters)

	go fetchKind0(relay, []string{}, eChan)

	go func() {
		for {
			select {
			case e := <-sendChan:
				if e.Kind == 15171031 {
					var accounts []string
					for _, tag := range e.Tags {
						if len(tag.Value()) == 64 {
							accounts = append(accounts, tag.Value())
						}
					}
					if len(accounts) > 0 {
						go fetchKind0(relay, accounts, sendChan)
					}
					continue
				}
				if e.Kind == 21069 {
					//fmt.Println("SENDING KEEPALIVE EVENT")
				}
				if e.Kind == 3340 {
					actors.LogCLI(fmt.Sprintf("publishing payment request event %s", e.ID), 4)
				}
				go func() {
					sane := library.ValidateSaneExecutionTime()
					_, err := relay.Publish(context.Background(), e)
					if err != nil {
						actors.LogCLI(err.Error(), 2)
					}
					sane()
					//library.LogCLI("Event "+e.ID+" publish status: "+status.String(), 4)
				}()
			}
		}
	}()

	go func() {
		<-sub.EndOfStoredEvents
		eose <- true
	}()
	lastEventTime := time.Now()
L:
	for {
		select {
		case <-sleepChan:
			go func() {
				actors.LogCLI("system sleep detected, terminating application", 2)
				cancel()
				actors.Shutdown()
			}()
		case ev := <-sub.Events:
			//fmt.Println(ev.ID)
			sane := library.ValidateSaneExecutionTime()
			if ev.Kind == 640001 {
			}
			if ev.Kind == 21069 {
				//fmt.Println("GOT KEEPALIVE EVENT")
			}
			if ev == nil {
				actors.LogCLI("Terminating connection to relay", 3)
				cancel()
				actors.LogCLI("Restarting Eventcatcher", 4)
				go SubscribeToTree(eChan, sendChan, eose)
				break L
			} else {
				go func() {
					lastEventTime = time.Now()
					if ev.Kind != 21069 { //ev.Kind >= 640000 && ev.Kind <= 649999 {
						if ok, _ := ev.CheckSignature(); ok {
							pushCache(*ev)
							eChan <- *ev
						}
					}
				}()
			}
			sane()
		case <-time.After(time.Minute):
			if time.Since(lastEventTime) > time.Duration(time.Minute*2) {
				go func() {
					actors.LogCLI("Terminating connection to relay", 3)
					cancel()
				}()
				actors.LogCLI("Restarting Eventcatcher", 4)
				go SubscribeToTree(eChan, sendChan, eose)
				break L
			}
			var t = nostr.Tags{}
			t = append(t, nostr.Tag{"e", actors.IgnitionEvent, "", "root"})
			keepAlive := nostr.Event{
				PubKey:    actors.MyWallet().Account,
				CreatedAt: time.Now(),
				Kind:      21069,
				Tags:      t,
			}

			keepAlive.ID = keepAlive.GetID()
			keepAlive.Sign(actors.MyWallet().PrivateKey)
			sendChan <- keepAlive
		case <-actors.GetTerminateChan():
			break L
		}
	}
	cancel()
}
