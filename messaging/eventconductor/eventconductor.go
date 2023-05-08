package eventconductor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/messaging/eventcatcher"
	"nostrocket/state/consensustree"
	"nostrocket/state/identity"
	"nostrocket/state/mirv"
	"nostrocket/state/replay"
	"nostrocket/state/shares"
)

type EventMap map[string]nostr.Event

var eventsInState = make(EventMap)
var eventsInStateLock = &deadlock.Mutex{}

func Start() {
	eventsInStateLock.Lock()
	eventsInState[actors.IgnitionEvent] = nostr.Event{}
	eventsInState[actors.StateChangeRequests] = nostr.Event{}
	eventsInState[actors.ReplayPrevention] = nostr.Event{}
	eventsInState[actors.ConsensusTree] = nostr.Event{}
	eventsInState[actors.Identity] = nostr.Event{}
	eventsInState[actors.Shares] = nostr.Event{}
	eventsInState[actors.Mirvs] = nostr.Event{}
	eventsInStateLock.Unlock()
	go handleEvents()
}

var started = make(map[string]bool)

var publishChan = make(chan nostr.Event)

func Publish(event nostr.Event) {
	go func() {
		publishChan <- event
		//fmt.Printf("\n48\n%#v\n", event)
	}()
}

var debug = false

func handleEvents() {
	if !started["handleEvents"] {
		started["handleEvents"] = true
		actors.GetWaitGroup().Add(1)
		var eoseChan = make(chan bool)
		var eventChan = make(chan nostr.Event)
		stack := library.NewEventStack(1)
		var eose bool
		go eventcatcher.SubscribeToTree(eventChan, publishChan, eoseChan)
		var timeToWaitBeforeHandlingNewStateChangeEvents time.Duration
		votepowerPosition := shares.GetPosition(actors.MyWallet().Account)
		if votepowerPosition > 0 {
			timeToWaitBeforeHandlingNewStateChangeEvents = time.Duration(votepowerPosition * 1000000 * 100)
		}
		lastReplayHash := replay.GetStateHash()
	L:
		for {
			select {
			case <-eoseChan:
				eose = true
			case event := <-eventChan:
				addEventToCache(event)
				if event.Kind == 640064 {
					//fmt.Printf("\nconsensus event from relay:\n%#v\n", event)
					err := handleConsensusEvent(event)
					if err != nil {
						library.LogCLI(err.Error(), 1)
					}
				} else {
					stack.Push(&event)
				}
			case <-time.After(timeToWaitBeforeHandlingNewStateChangeEvents):
				if debug {
					continue
				}
				lastReplayHash = replay.GetStateHash()
				//todo create exception for ignition event if we are ignition account
				if eose && shares.VotepowerForAccount(actors.MyWallet().Account) > 0 && votepowerPosition > 0 {
					for _, event := range consensustree.DeleteDuplicateConsensusEvents() {
						fmt.Printf("\nTODO: publish here after we know we aren't deleting valid events\n%#v\n", event)
					}
					event, ok := stack.Pop()
					if ok {
						processStateChangeEventOutOfConsensus(event)
					}
					if !ok {
						if replay.GetStateHash() != lastReplayHash {
							library.LogCLI("Some state has changed, so we are attempting to replay previously failed state change events", 4)
							lastReplayHash = replay.GetStateHash()
							for _, n := range getAllUnhandledStateChangeEventsFromCache() {
								processStateChangeEventOutOfConsensus(&n)
							}
						}
					}
				}

			case <-actors.GetTerminateChan():
				//just keeping this here for shutdown hooks
				actors.GetWaitGroup().Done()
				break L
			}
		}
	}
}

func processStateChangeEventOutOfConsensus(event *nostr.Event) {
	err := handleEvent(*event, false)
	if err != nil {
		//library.LogCLI(err.Error(), 2)
	}
	if err == nil {
		err = consensustree.CreateNewConsensusEvent(*event, publishChan)
		if err != nil {
			library.LogCLI(err.Error(), 1)
		}
	}
}

func handleConsensusEvent(e nostr.Event) error {
	toHandle := make(chan library.Sha256)
	consensusEventsToPublish := make(chan nostr.Event)
	var returnResult = make(chan bool)
	go func() {
		for {
			select {
			case e := <-consensusEventsToPublish:
				fmt.Printf("\nCONSENSUS EVENT TO PUBLISH:\n%#v\n", e)
				Publish(e)
			case e := <-toHandle:
				event, ok := getEventFromCache(e)
				if !ok {
					library.LogCLI("could not get event "+e, 2)
					returnResult <- false
				}
				if ok {
					if err := handleEvent(event, true); err != nil {
						library.LogCLI(fmt.Sprintf("%s failed: %s", event.ID, err.Error()), 1)
						returnResult <- false
					} else {
						returnResult <- true
					}
				}
			case <-time.After(time.Second * 6):
				return
			}
		}
	}()
	return consensustree.HandleConsensusEvent(e, toHandle, returnResult, consensusEventsToPublish)
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

func GetEventFromCache(id library.Sha256) (n nostr.Event) {
	if e, ok := getEventFromCache(id); ok {
		n = e
	}
	return
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

func getAllUnhandledStateChangeEventsFromCache() (el []nostr.Event) {
	eventCacheWg.Wait()
	eventCacheMu.Lock()
	defer eventCacheMu.Unlock()
	for _, event := range eventCache {
		if event.Kind != 640064 {
			if !eventIsInState(event.ID) {
				el = append(el, event)
			}
		}
	}
	return
}

//func processEvent(e nostr.Event) error {
//	eventsInStateLock.Lock()
//	defer eventsInStateLock.Unlock()
//	if eventIsInState(e.ID) {
//		return nil
//	}
//	if eventsInState.isDirectReply(e) {
//		err := handleEvent(e, false)
//		if err != nil {
//			return err
//		}
//		return nil
//	}
//	return fmt.Errorf("event is not in direct reply to any other event in nostrocket state")
//}

func handleEvent(e nostr.Event, fromConsensusEvent bool) error {
	if eventIsInState(e.ID) {
		return fmt.Errorf("event %s is already in our local state", e.ID)
	}
	library.LogCLI(fmt.Sprintf("Attempting to handle state change event %s [consensus mode: %v]", e.ID, fromConsensusEvent), 4)
	closer, returner, ok := replay.HandleEvent(e)
	if ok {
		eventsInState[e.ID] = e
		//fmt.Printf("\n---HANDLING EVENT---\n%#v\n--------\n", e)
		mindName, mappedState, err := routeEvent(e)
		if err != nil {
			closer <- false
			close(returner)
			return err
		} else {
			closer <- true
			mappedReplay := <-returner
			close(returner)
			library.LogCLI(fmt.Sprintf("State has been updated by %s [consensus mode: %v]", e.ID, fromConsensusEvent), 3)
			actors.AppendState("replay", mappedReplay)
			n, _ := actors.AppendState(mindName, mappedState)
			b, err := json.Marshal(n)
			if err != nil {
				return err
			}
			if err == nil {
				//publish our current state
				//todo only publish if we are at the current bitcoin tip
				if !fromConsensusEvent {
					Publish(actors.CurrentStateEventBuilder(fmt.Sprintf("%s", b)))
				}
				return nil
			}
		}
	}
	return fmt.Errorf("invalid replay")
}

func eventIsInState(e library.Sha256) bool {
	eventsInStateLock.Lock()
	defer eventsInStateLock.Unlock()
	_, exists := eventsInState[e]
	return exists
}

func routeEvent(e nostr.Event) (mindName string, newState any, err error) {
	//todo get current state from each Mind, and verify that state was actually changed, return error if not
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
		mindName = "mirvs"
		newState, err = mirv.HandleEvent(e)
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
