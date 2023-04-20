package eventconductor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/consensus/consensustree"
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
	eventsInState[actors.ConsensusTree] = nostr.Event{}
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
		//fmt.Printf("\n48\n%#v\n", event)
	}()
}

func handleEvents() {
	if !started {
		started = true
		actors.GetWaitGroup().Add(1)
		var eose = make(chan bool)
		go eventcatcher.SubscribeToTree(eventChan, sendChan, eose)
		var toReplay []nostr.Event
		var reachedEose = false

	L:
		for {
			select {
			case e := <-eventChan:
				if !reachedEose {
					go addEventToCache(e)
				}
				toReplay = append(toReplay, e)
				//processEvent(e, &toReplay)
			//if event is in direct reply to an event that is in state, try to handle it. if not, put it aside to try again later
			//if we are at the current tip, then when we see a new block from a block source, tag all current leaf nodes
			//if we are not at the current tip, it means we are in catchup mode, so when a mind thread hits a block tag, pause until global state reaches that block.
			case <-eose:
				reachedEose = true
				eventCacheWg.Wait()
				toHandle := make(chan library.Sha256)
				waitForConsensus := &deadlock.WaitGroup{}
				var returnResult = make(chan bool)
				go consensustree.HandleBatchAfterEOSE(getAll640064(), waitForConsensus, toHandle, returnResult)
				go func() {
					select {
					case e := <-toHandle:
						event, ok := getEventFromCache(e)
						if !ok {
							library.LogCLI("could not get event "+e, 2)
							returnResult <- false
						}
						if ok {
							if handleEvent(event) != nil {
								fmt.Printf("\n%#v\n", event)
								library.LogCLI("consenus inner event failed", 1)
								returnResult <- false
							} else {
								returnResult <- true
							}
						}

					}
				}()
				//rebuild state from the consensus log
				//send all the 640064 events to consensustree
				//if height == currentheight+1 process
				//use a callback channel to request handling of embedded event, terminate if fail
				//if > currentheight+1
				//if current height and event ID have >500 permille, process embedded event, otherwise wait for more consensus events, unless we have votepower in which case: ...

			case <-time.After(time.Second * 10):
				var replayTemp []nostr.Event
				for _, event := range toReplay {
					processEvent(event, &replayTemp)
				}
				toReplay = []nostr.Event{}
				toReplay = replayTemp
			case <-actors.GetTerminateChan():
				for i, event := range toReplay {
					fmt.Printf("\nReplay Queue %d%s\n", i, event.ID)
				}
				actors.GetWaitGroup().Done()
				break L
			}
		}
	}
}

var eventCache = make(map[library.Sha256]nostr.Event)
var eventCacheMu = &deadlock.Mutex{}
var eventCacheWg = &deadlock.WaitGroup{}

func addEventToCache(e nostr.Event) {
	eventCacheWg.Add(1)
	eventCacheMu.Lock()
	eventCache[e.ID] = e
	eventCacheMu.Unlock()
	eventCacheWg.Done()
}

func getEventFromCache(eventID library.Sha256) (nostr.Event, bool) {
	eventCacheWg.Wait()
	eventCacheMu.Lock()
	defer eventCacheMu.Unlock()
	e, ok := eventCache[eventID]
	return e, ok
}

func getAll640064() (el []nostr.Event) {
	//todo filter so we only reutrn unique inner event + signer + height
	eventCacheMu.Lock()
	defer eventCacheMu.Unlock()
	for _, event := range eventCache {
		if event.Kind == 640064 {
			el = append(el, event)
		}
	}
	return
}

var printed = make(map[string]struct{})

func processEvent(e nostr.Event, toReplay *[]nostr.Event) {
	eventsInStateLock.Lock()
	defer eventsInStateLock.Unlock()
	if _, exists := printed[e.ID]; !exists {
		//fmt.Printf("\n80: %s\n", e.ID)
		printed[e.ID] = struct{}{}
	}
	if eventsInState.isDirectReply(e) {
		err := handleEvent(e)
		if err != nil {
			library.LogCLI(err.Error(), 1)
			//todo do we need to replay here?
		}
	} else {
		*toReplay = append(*toReplay, e)
		//fmt.Println("TO REPLAY: ", e.ID)
	}
}

func handleEvent(e nostr.Event) error {
	closer, returner, ok := replay.HandleEvent(e)
	if ok {
		eventsInState[e.ID] = e
		fmt.Printf("\n---HANDLING EVENT---\n%#v\n--------\n", e)
		mindName, mappedState, err := routeEvent(e)
		if err != nil {
			closer <- false
			close(returner)
			return err
		} else {
			closer <- true
			mappedReplay := <-returner
			close(returner)
			publishConsensusTree(e)
			actors.AppendState("replay", mappedReplay)
			n, _ := actors.AppendState(mindName, mappedState)
			b, err := json.Marshal(n)
			if err != nil {
				return err
			} else {
				fmt.Printf("\n199: %s", b)
				Publish(actors.EventBuilder(fmt.Sprintf("%s", b)))
				return nil
			}
		}
	}
	return fmt.Errorf("invalid replay")
}

func publishConsensusTree(e nostr.Event) {
	if shares.VotepowerForAccount(actors.MyWallet().Account) > 0 {
		//todo get current bitcoin height

		consensusEvent, err := consensustree.ProduceEvent(e.ID, 0)
		if err != nil {
			library.LogCLI(err, 1)
			return
		}
		_, _, err = consensustree.HandleEvent(consensusEvent)
		if err != nil {
			library.LogCLI(err.Error(), 0)
		} else {
			fmt.Printf("\n221: %#v\n", consensusEvent)
			Publish(consensusEvent)
		}
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
	case k == 640064:
		fmt.Printf("\n640064\n%#v\n", e)
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
