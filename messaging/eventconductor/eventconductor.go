package eventconductor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/consensus/identity"
	"nostrocket/consensus/replay"
	"nostrocket/consensus/shares"
	"nostrocket/consensus/subrockets"
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
	eventsInState[actors.StateChangeRequests] = nostr.Event{}
	eventsInState[actors.ReplayPrevention] = nostr.Event{}
	eventsInState[actors.Identity] = nostr.Event{}
	eventsInState[actors.Shares] = nostr.Event{}
	eventsInState[actors.Subrockets] = nostr.Event{}
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
		go eventcatcher.SubscribeToTree(eventChan, sendChan)
		var toReplay []nostr.Event
	L:
		for {
			select {
			case e := <-eventChan:
				processEvent(e, &toReplay)
			//if event is in direct reply to an event that is in state, try to handle it. if not, put it aside to try again later
			//if we are at the current tip, then when we see a new block from a block source, tag all current leaf nodes
			//if we are not at the current tip, it means we are in catchup mode, so when a mind thread hits a block tag, pause until global state reaches that block.
			case <-time.After(time.Second * 5):
				var replayTemp []nostr.Event
				for _, event := range toReplay {
					processEvent(event, &replayTemp)
				}
				toReplay = []nostr.Event{}
				toReplay = replayTemp
			case <-actors.GetTerminateChan():
				for _, event := range toReplay {
					fmt.Printf("\n%#v\n", event)
				}
				actors.GetWaitGroup().Done()
				break L
			}
		}
	}
}

var printed = make(map[string]struct{})

func processEvent(e nostr.Event, toReplay *[]nostr.Event) {
	eventsInStateLock.Lock()
	defer eventsInStateLock.Unlock()
	if _, exists := printed[e.ID]; !exists {
		fmt.Printf("\n80: %s\n", e.ID)
		printed[e.ID] = struct{}{}
	}
	if eventsInState.isDirectReply(e) {
		if closer, returner, ok := replay.HandleEvent(e); ok {
			eventsInState[e.ID] = e
			fmt.Printf("\n---HANDLING EVENT---\n%#v\n--------\n", e)
			mindName, mappedState, err := routeEvent(e)
			if err != nil {
				library.LogCLI(err.Error(), 2)
				closer <- false
				close(returner)
			} else {
				closer <- true
				mappedReplay := <-returner
				close(returner)
				actors.AppendState("replay", mappedReplay)
				n, _ := actors.AppendState(mindName, mappedState)
				b, err := json.Marshal(n)
				if err != nil {
					library.LogCLI(err.Error(), 1)
				} else {
					fmt.Printf("%s", b)
					Publish(actors.EventBuilder(fmt.Sprintf("%s", b)))
				}
			}
		}
	} else {
		*toReplay = append(*toReplay, e)
		//fmt.Println("TO REPLAY: ", e.ID)
	}
}

func routeEvent(e nostr.Event) (mindName string, newState any, err error) {
	switch k := e.Kind; {
	default:
		mindName = ""
		newState = nil
		err = fmt.Errorf("no mind to handle kind %s", e.Kind)
	case k >= 640400 && k <= 640499:
		mindName = "identity"
		newState, err = identity.HandleEvent(e)
	case k >= 640200 && k <= 640299:
		mindName = "shares"
		newState, err = shares.HandleEvent(e)
	case k >= 640600 && k <= 640699:
		mindName = "subrockets"
		newState, err = subrockets.HandleEvent(e)
	}
	return
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
						return true
					}
				}
			}
		}

	}
	return false
}
