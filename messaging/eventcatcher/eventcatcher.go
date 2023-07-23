package eventcatcher

import (
	"context"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/prashantgupta24/mac-sleep-notifier/notifier"
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
}

func FetchCache(id string) (e *nostr.Event, r bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	ev, re := cache[id]
	return &ev, re
}

func SubscribeToTree(eChan chan nostr.Event, sendChan chan nostr.Event, eose chan bool) {
	sleepChan := notifier.GetInstance().Start() //sleep.GetInstance().Start() //sleep.PingWhenSleep()
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
	actors.LogCLI("Connecting to "+relay.URL, 4)
	sub := relay.Subscribe(ctx, filters)

	go func() {
		for {
			select {
			case e := <-sendChan:
				if e.Kind == 21069 {
					//fmt.Println("SENDING KEEPALIVE EVENT")
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
