package relays

import (
	"context"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

var sendToConductor chan nostr.Event

func SubscribeToIgnitionTree(eChan chan nostr.Event, sendChan chan nostr.Event, eose chan bool) {
	var sleepChan = make(chan bool)
	sleeper(sleepChan)
	sendToConductor = eChan
	mainRelay, err := nostr.RelayConnect(context.Background(), "wss://nostr.688.org")
	if err != nil {
		actors.LogCLI(fmt.Sprintf("could not connect to relay: %s", err), 0)
	}
	tags := make(map[string][]string)
	tags["e"] = []string{actors.IgnitionEvent}
	var filters nostr.Filters
	filters = []nostr.Filter{{
		Tags: tags,
	}}

	ctx, cancel := context.WithCancel(context.Background())
	actors.LogCLI("Connecting to "+mainRelay.URL, 4)
	sub, err := mainRelay.Subscribe(ctx, filters)
	if err != nil {
		actors.LogCLI(err.Error(), 1)
	}
	lastEventTime := time.Now()
	go func() {
		<-sub.EndOfStoredEvents
		eose <- true
	}()
L:
	for {
		select {
		case <-sleepChan:
			go func() {
				actors.LogCLI("system sleep detected, terminating application", 2)
				cancel()
				actors.Shutdown()
			}()
		case e := <-sendChan:
			if e.Kind == 21069 {
				//fmt.Println("SENDING KEEPALIVE EVENT")
			}
			if e.Kind == 3340 {
				actors.LogCLI(fmt.Sprintf("publishing payment request event %s", e.ID), 4)
			}
			go func() {
				sane := library.ValidateSaneExecutionTime()
				if ok, _ := e.CheckSignature(); ok {
					_, err := mainRelay.Publish(context.Background(), e)
					if err != nil {
						actors.LogCLI(err.Error(), 1)
						go func() { sendChan <- e }()
					}
				} else {
					fmt.Printf("\n%#v\n", e)
				}
				sane()
				//library.LogCLI("Event "+e.ID+" publish status: "+status.String(), 4)
			}()
		case ev := <-sub.Events:
			//fmt.Println(ev.ID)
			sane := library.ValidateSaneExecutionTime()
			if ev.Kind == 21069 {
				//fmt.Println("GOT KEEPALIVE EVENT")
			}
			if ev == nil {
				actors.LogCLI("Terminating connection to relay", 3)
				cancel()
				mainRelay.Close()
				actors.LogCLI("Restarting Eventcatcher", 4)
				go SubscribeToIgnitionTree(eChan, sendChan, eose)
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
			if time.Since(lastEventTime) > time.Minute*2 {
				go func() {
					actors.LogCLI("Terminating connection to relay", 3)
					cancel()
					mainRelay.Close()
				}()
				actors.LogCLI("Restarting Eventcatcher", 4)
				go SubscribeToIgnitionTree(eChan, sendChan, eose)
				break L
			}
			go func() { sendChan <- makeKeepAliveEvent() }()
		case <-actors.GetTerminateChan():
			break L
		}
	}
	cancel()
	mainRelay.Close()
}

func makeKeepAliveEvent() nostr.Event {
	var t = nostr.Tags{}
	t = append(t, nostr.Tag{"e", actors.IgnitionEvent, "", "root"})
	keepAlive := nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      21069,
		Tags:      t,
	}

	keepAlive.ID = keepAlive.GetID()
	keepAlive.Sign(actors.MyWallet().PrivateKey)
	return keepAlive
}
