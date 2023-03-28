package eventconductor

import (
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/consensus/identity"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/messaging/eventcatcher"
)

type EventMap map[string]nostr.Event

var eventsInState = make(EventMap)
var eventsInStateLock = &deadlock.Mutex{}

func Start() {
	//todo load events in state from current state
	eventsInStateLock.Lock()
	eventsInState[actors.IgnitionEvent] = nostr.Event{}
	eventsInStateLock.Unlock()
	go handleEvents()
}

var eventChan = make(chan nostr.Event)
var started = false

var sendChan = make(chan nostr.Event)

func Publish(event nostr.Event) {
	go func() {
		sendChan <- event
	}()
}

func handleEvents() {
	if !started {
		started = true
		actors.GetWaitGroup().Add(1)
		terminateChan := actors.GetTerminateChan()
		go eventcatcher.SubscribeToTree(terminateChan, eventChan, sendChan)
		var replay []nostr.Event
	L:
		for {
			select {
			case e := <-eventChan:
				processEvent(e, &replay)
			//if event is in direct reply to an event that is in state, try to handle it. if not, put it aside to try again later
			//if we are at the current tip, then when we see a new block from a block source, tag all current leaf nodes
			//if we are not at the current tip, it means we are in catchup mode, so when a mind thread hits a block tag, pause until global state reaches that block.
			case <-time.After(time.Second * 5):
				var replayTemp []nostr.Event
				fmt.Println("49")
				for _, event := range replay {
					processEvent(event, &replayTemp)
				}
				replay = []nostr.Event{}
				replay = replayTemp
			case <-terminateChan:
				for _, event := range replay {
					fmt.Printf("\n%#v\n", event.Tags)
				}
				actors.GetWaitGroup().Done()
				break L
			}
		}
	}
}

func processEvent(e nostr.Event, replay *[]nostr.Event) {
	eventsInStateLock.Lock()
	defer eventsInStateLock.Unlock()
	if eventsInState.isDirectReply(e) {
		//todo handle event with appropriate mind
		eventsInState[e.ID] = e
		fmt.Println("DIRECT REPLY: ", e.ID)
		if e.Kind == 640400 {
			m, ok := identity.HandleEvent(e)
			if !ok {
				library.LogCLI("error", 1)
			} else {
				for account, i := range m {
					fmt.Printf("ACCOUNT: %s\n%#v\n", account, i)
				}
			}
		}
	} else {
		*replay = append(*replay, e)
		//fmt.Println("TO REPLAY: ", e.ID)
	}
}

func (m EventMap) isDirectReply(event nostr.Event) bool {
	for _, tag := range event.Tags {
		if len(tag) >= 2 {
			if tag[0] == "e" {
				if _, exists := m[tag[1]]; exists {
					if len(tag) > 3 {
						if tag[3] != "root" {
							return true
						}
					} else {
						fmt.Println(86)
						return true
					}
				}
			}
		}

	}
	return false
}
