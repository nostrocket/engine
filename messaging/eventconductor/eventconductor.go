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

var started = false

var sendChan = make(chan nostr.Event)

func Publish(event nostr.Event) {
	go func() {
		sendChan <- event
		//fmt.Printf("\n48\n%#v\n", event)
	}()
}

func sendToCache(e chan nostr.Event, liveModeChan, liveModeConsensusChan chan nostr.Event) {
	for {
		select {
		case event := <-e:
			go func() {
				if event.Kind == 640064 {
					liveModeConsensusChan <- event
				} else {
					liveModeChan <- event
				}
			}()
			addEventToCache(event)
		}
	}
}

func handleEvents() {
	if !started {
		started = true
		actors.GetWaitGroup().Add(1)
		var eose = make(chan bool)
		var eventChan = make(chan nostr.Event)
		var liveModeStateChangeChan = make(chan nostr.Event)
		var liveModeConsensusChan = make(chan nostr.Event)
		go sendToCache(eventChan, liveModeStateChangeChan, liveModeConsensusChan)
		go eventcatcher.SubscribeToTree(eventChan, sendChan, eose)
		var toReplay []nostr.Event
	L:
		for {
			select {
			case <-eose:
				eventCacheWg.Wait()
				toHandle := make(chan library.Sha256)
				consensusEventsToPublish := make(chan nostr.Event)
				var returnResult = make(chan bool)
				go func() {
					for {
						select {
						case e := <-consensusEventsToPublish:
							fmt.Printf("\nCONSENSUS EVENT TO PUBLISH:\n%#v\n", e)
							//Publish(e)
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
						case <-actors.GetTerminateChan():
							return
						}
					}
				}()
				consensustree.HandleBatchAfterEOSE(getAll640064(), toHandle, consensusEventsToPublish, returnResult)

				go handleEventsInLiveMode(liveModeStateChangeChan, liveModeConsensusChan)
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

var errors = make(map[library.Sha256]int64)

func stopTryingThisEvent(e library.Sha256) bool {
	num, exists := errors[e]
	if exists {
		if num > 10 {
			return true
		}
	}
	return false
}

func handleEventsInLiveMode(stateChange, consensus chan nostr.Event) {
	//todo handle consensus events in live mode
	for {
		select {
		case <-actors.GetTerminateChan():
			return
		case e := <-consensus:
			err := consensustree.HandleEvent(e)
			if err != nil {
				library.LogCLI(err.Error(), 1)
			}
		case e := <-stateChange:
			if stopTryingThisEvent(e.ID) {
				continue
			}
			err := processEvent(e)
			if err != nil {
				library.LogCLI(err.Error(), 1)
				errors[e.ID]++
				go func() {
					time.Sleep(time.Millisecond * 500)
					stateChange <- e
				}()
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

func getAllStateChangeEventsFromCache() (el []nostr.Event) {
	eventCacheWg.Wait()
	eventCacheMu.Lock()
	defer eventCacheMu.Unlock()
	for _, event := range eventCache {
		if event.Kind != 640064 {
			el = append(el, event)
		}
	}
	return
}

func processEvent(e nostr.Event) error {
	eventsInStateLock.Lock()
	defer eventsInStateLock.Unlock()
	if eventIsInState(e.ID) {
		return nil
	}
	if eventsInState.isDirectReply(e) {
		err := handleEvent(e, false)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("event is not in direct reply to any other event in nostrocket state")
}

func handleEvent(e nostr.Event, catchupMode bool) error {
	if eventIsInState(e.ID) {
		return fmt.Errorf("event %s is already in our local state", e.ID)
	}
	library.LogCLI(fmt.Sprintf("Attempting to handle state change event %s catchup mode: %v", e.ID, catchupMode), 4)
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
			library.LogCLI(fmt.Sprintf("Handled state change event %s catchup mode: %v", e.ID, catchupMode), 4)
			if !catchupMode {
				publishConsensusTree(e)
			}
			actors.AppendState("replay", mappedReplay)
			n, _ := actors.AppendState(mindName, mappedState)
			b, err := json.Marshal(n)
			if err != nil {
				return err
			}
			if err == nil {
				//publish our current state
				Publish(actors.CurrentStateEventBuilder(fmt.Sprintf("%s", b)))
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

func publishConsensusTree(e nostr.Event) {
	if shares.VotepowerForAccount(actors.MyWallet().Account) > 0 {
		//todo get current bitcoin height

		consensusEvent, err := consensustree.ProduceEvent(e.ID, 0)
		if err != nil {
			library.LogCLI(err, 1)
			return
		}
		err = consensustree.HandleEvent(consensusEvent)
		if err != nil {
			library.LogCLI(err.Error(), 0)
		} else {
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
