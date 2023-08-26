package eventconductor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"golang.org/x/exp/slices"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/messaging/blocks"
	"nostrocket/messaging/relays"
	"nostrocket/state/consensustree"
	"nostrocket/state/identity"
	"nostrocket/state/merits"
	"nostrocket/state/payments"
	"nostrocket/state/problems"
	"nostrocket/state/replay"
	"nostrocket/state/rockets"
)

// todo don't handle out of consensus events unless our height is > x
// (not sure how to calculate x yet, but should be something that prevents premature emission of consensus events when something weird happens)
type EventMap map[string]nostr.Event

var bitcoinBlocks []bblocks
var highestStateChangeEventSeen int64

type bblocks struct {
	height int64
	event  nostr.Event
}

var eventsInState = make(EventMap)
var eventsInStateLock = &deadlock.Mutex{}

func Start() {
	eventsInStateLock.Lock()
	eventsInState[actors.IgnitionEvent] = nostr.Event{}
	eventsInState[actors.StateChangeRequests] = nostr.Event{}
	eventsInState[actors.ReplayPrevention] = nostr.Event{}
	eventsInState[actors.ConsensusTree] = nostr.Event{}
	eventsInState[actors.Identity] = nostr.Event{}
	eventsInState[actors.Merits] = nostr.Event{}
	eventsInState[actors.Rockets] = nostr.Event{}
	eventsInState[actors.Problems] = nostr.Event{}
	eventsInStateLock.Unlock()
	go handleEvents()
}

var started = make(map[string]bool)

var publishChan = make(chan nostr.Event)

func Publish(event nostr.Event) {
	go func() {
		sane := library.ValidateSaneExecutionTime()
		defer sane()
		//if event.Kind == 15171010 {
		//	fmt.Printf("\n%#v\n", event)
		//}
		go func() { publishChan <- event }()
		//fmt.Printf("\n51%#v\n", event)
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
		go relays.SubscribeToIgnitionTree(eventChan, publishChan, eoseChan)
		var timeToWaitBeforeHandlingNewStateChangeEvents time.Duration
		votepowerPosition := merits.GetPosition(actors.MyWallet().Account)
		if votepowerPosition > 0 {
			//hacky way to avoid consensus reorgs - let the highest votepower go first.
			timeToWaitBeforeHandlingNewStateChangeEvents = time.Duration(votepowerPosition * 1000000 * 100)
		}
		lastReplayHash := replay.GetStateHash()
		var timeSinceLastKind0Query time.Time
	L:
		for {
			select {
			case <-eoseChan:
				eose = true
			case event := <-eventChan:
				if !addEventToCache(event) {
					if event.Kind == 640001 && !eventIsInState(event.ID) {
						actors.LogCLI(fmt.Sprintf("consensus event from relay: %s", event.ID), 4)
						err := handleConsensusEvent(event)
						if err != nil {
							actors.LogCLI(err.Error(), 0)
						}
					} else {
						if event.Kind == 1517 {
							if heightStr, ok := library.GetFirstTag(event, "height"); ok {
								if height, err := strconv.ParseInt(heightStr, 10, 64); err == nil {
									if height > blocks.Tip().Height {
										bitcoinBlocks = append(bitcoinBlocks, bblocks{
											height: height,
											event:  event,
										})
										sort.Slice(bitcoinBlocks, func(i, j int) bool {
											return bitcoinBlocks[i].height > bitcoinBlocks[j].height
										})
									}

								}
							}
							continue
						}
						if event.Kind == 10311 {
							if event.PubKey == actors.MyWallet().Account {
								if consensusHeight, ok := library.GetFirstTag(event, "consensus"); ok {
									parsedInt, err := strconv.ParseInt(consensusHeight, 10, 64)
									if err == nil {
										highestStateChangeEventSeen = parsedInt
									}
								}
							}
							continue
						}
						stack.Push(&event)
					}
				}
			case <-time.After(timeToWaitBeforeHandlingNewStateChangeEvents):
				if debug {
					continue
				}
				lastReplayHash = replay.GetStateHash()
				//todo create exception for ignition event if we are ignition account
				if eose && merits.VotepowerInNostrocketForAccount(actors.MyWallet().Account) > 0 && votepowerPosition > 0 {
					for _, event := range consensustree.DeleteDuplicateConsensusEvents() {
						fmt.Printf("\nTODO: publish here after we know we aren't deleting valid events\n%#v\n", event)
					}
					event, ok := stack.Pop()
					if ok {
						for _, block := range bitcoinBlocks {
							if block.height > blocks.Tip().Height {
								if err := processStateChangeEventOutOfConsensus(&block.event); err == nil {
									break
								}
							}
						}
						bitcoinBlocks = []bblocks{}
						err := processStateChangeEventOutOfConsensus(event)
						if err != nil {
							//actors.LogCLI(fmt.Sprintf("event %s: ", err.Error()), 2)
						}
					}
					if !ok {
						if time.Since(timeSinceLastKind0Query) > time.Minute*10 {
							timeSinceLastKind0Query = time.Now()
							for _, account := range subscribeToKind0Events() {
								go relays.FetchLatestProfile(account)
							}
						}
						if replay.GetStateHash() != lastReplayHash {
							actors.LogCLI("Some state has changed, so we are attempting to replay previously failed state change events", 4)
							lastReplayHash = replay.GetStateHash()
							for _, n := range getAllUnhandledStateChangeEventsFromCache() {
								err := processStateChangeEventOutOfConsensus(&n)
								if err != nil {
									//actors.LogCLI(fmt.Sprintf("event %s: ", err.Error()), 2)
								}
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

var replayExceptions = []int{0, 9735, 1517}
var consensusExceptions = []int{0, 9735}

func processStateChangeEventOutOfConsensus(event *nostr.Event) error {
	sane := library.ValidateSaneExecutionTime()
	defer sane()
	if time.Since(event.CreatedAt.Time()) > time.Hour*24 && !slices.Contains(replayExceptions, event.Kind) {
		return fmt.Errorf("we are probably missing a consensus event")
	}
	err := handleEvent(*event, false)
	newConsensusEventNeeded := int64(len(consensustree.GetAllStateChangeEventsInOrder())) >= highestStateChangeEventSeen
	if err == nil && !slices.Contains(consensusExceptions, event.Kind) && newConsensusEventNeeded {
		consensusEvent, err := consensustree.CreateNewConsensusEvent(*event)
		if err != nil {
			return err
		}
		err = consensustree.HandleConsensusEvent(consensusEvent, nil, nil, nil, true)
		if err != nil {
			return err
		}
		if err == nil {
			addEventToState(consensusEvent.ID)
			Publish(consensusEvent)
			actors.LogCLI(fmt.Sprintf("published consensus event for state change caused by %s", event.ID), 3)
		}
	}
	return err
}

func handleConsensusEvent(e nostr.Event) error {
	sane := library.ValidateSaneExecutionTime()
	defer sane()
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
					time.Sleep(time.Millisecond * 500)
					event, ok = getEventFromCache(e)
					if !ok {
						ev, fetchok := relays.FetchCache(e)
						event = *ev
						if !fetchok {
							actors.LogCLI("could not get event "+e, 2)
							returnResult <- false
						}
						ok = true
					}
				}
				if ok {
					if err := handleEvent(event, true); err != nil {
						actors.LogCLI(fmt.Sprintf("%s failed: %s", event.ID, err.Error()), 1)
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
	return consensustree.HandleConsensusEvent(e, toHandle, returnResult, consensusEventsToPublish, false)
}

var eventCache = make(map[library.Sha256]nostr.Event)
var eventCacheMu = &deadlock.Mutex{}
var eventCacheWg = &deadlock.WaitGroup{}

func addEventToCache(e nostr.Event) bool {
	eventCacheWg.Add(1)
	eventCacheMu.Lock()
	_, exists := eventCache[e.ID]
	eventCache[e.ID] = e
	eventCacheMu.Unlock()
	eventCacheWg.Done()
	return exists
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

func getAll640001() (el []nostr.Event) {
	//todo filter so we only reutrn unique inner event + signer + height
	eventCacheMu.Lock()
	defer eventCacheMu.Unlock()
	for _, event := range eventCache {
		if event.Kind == 640001 {
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
		if event.Kind != 640001 {
			if !eventIsInState(event.ID) {
				el = append(el, event)
			}
		}
	}
	return
}

func handleOutbox(state any) (mindName string, newMappedState any, exists bool) {
	switch e := state.(type) {
	case payments.Mapped:
		for _, outbox := range e.Outbox {
			switch o := outbox.(type) {
			case nostr.Event:
				switch o.Kind {
				case 15179735:
					Publish(o)
				case 15173340:
					Publish(o)
				case 15171010:
					events, relayUrls := payments.GetAuthEvents()
					relays.PublishToRelays(events, relayUrls)
				case 15171031:
					if len(o.Content) == 64 {
						relays.Subscribe(o.Content)
					}
				default:
					actors.LogCLI("unhandled event type in outbox", 1)
					fmt.Printf("\n%#v\n", o)
				}
			case merits.Mapped:
				mindName = "merits"
				newMappedState = o
				exists = true
			}

		}
	}
	return
}

func handleEvent(e nostr.Event, fromConsensusEvent bool) error {
	if eventIsInState(e.ID) {
		return fmt.Errorf("event %s is already in our local state", e.ID)
	}
	actors.LogCLI(fmt.Sprintf("Attempting to handle state change event %s kind %d [consensus mode: %v]", e.ID, e.Kind, fromConsensusEvent), 4)
	closer, replayState, replayOk := replay.HandleEvent(e)
	if !replayOk && !slices.Contains(replayExceptions, e.Kind) {
		return fmt.Errorf("invalid replay")
	}
	eventsInState[e.ID] = e
	e.SetExtra("fromConsensusEvent", fromConsensusEvent)
	mindName, mappedState, err := routeEvent(e)
	if err != nil {
		if replayOk {
			closer <- false
			close(replayState)
		}
		actors.LogCLI(err.Error(), 3)
		return err
	}
	actors.LogCLI(fmt.Sprintf("State of %s has been updated by %s [consensus mode: %v]", mindName, e.ID, fromConsensusEvent), 3)
	if !slices.Contains(replayExceptions, e.Kind) {
		closer <- true
		newReplayState := <-replayState
		close(replayState)
		AppendState("replay", newReplayState)
	}
	if mn, ns, ok := handleOutbox(mappedState); ok {
		AppendState(mn, ns)
	}
	n, _ := AppendState(mindName, mappedState)
	b, err := json.Marshal(n)
	if err != nil {
		return err
	}
	//publish our current state
	//todo only publish if we are at the current bitcoin tip
	if !fromConsensusEvent && !slices.Contains(replayExceptions, e.Kind) {
		stateEvent := CurrentStateEventBuilder(fmt.Sprintf("%s", b))
		Publish(stateEvent)
		actors.LogCLI(fmt.Sprintf("Published current state in event %s", stateEvent.ID), 4)
		time.Sleep(time.Second)
	}
	return nil
}

func eventIsInState(e library.Sha256) bool {
	eventsInStateLock.Lock()
	defer eventsInStateLock.Unlock()
	_, exists := eventsInState[e]
	return exists
}

func addEventToState(e library.Sha256) {
	eventsInStateLock.Lock()
	defer eventsInStateLock.Unlock()
	eventsInState[e] = nostr.Event{}
}

func routeEvent(e nostr.Event) (mindName string, newState any, err error) {
	//todo get current state from each Mind, and verify that state was actually changed, return error if not
	switch k := e.Kind; {
	default:
		mindName = ""
		newState = nil
		err = fmt.Errorf("no mind to handle kind %d", e.Kind)
	case k == 1517:
		mindName = "blocks"
		newState, err = blocks.HandleEvent(e)
	case k >= 640400 && k <= 640499 || k == 0:
		mindName = "identity"
		newState, err = identity.HandleEvent(e)
	case k >= 640200 && k <= 640299:
		mindName = "merits"
		newState, err = merits.HandleEvent(e)
	case k >= 640600 && k <= 640699:
		mindName = "rockets"
		newState, err = rockets.HandleEvent(e)
	case k >= 641800 && k <= 641899:
		mindName = "problems"
		newState, err = problems.HandleEvent(e)
	case k == 640001:
		fmt.Printf("\n640001\n%#v\n", e)
	case k == 3340 || k == 15173340 || k == 9735 || k == 15179735:
		mindName = "payments"
		newState, err = payments.HandleEvent(e)
	case k == 1:
		mindName = ""
		newState = nil
		err = fmt.Errorf("unhandled opcode on kind 1 event")
		if operation, ok := library.GetFirstTag(e, "op"); ok {
			ops := strings.Split(operation, ".")
			if len(ops) > 2 {
				if ops[0] == "nostrocket" {
					switch o := ops[1]; {
					case o == "problem":
						mindName = "problems"
						newState, err = problems.HandleEvent(e)
					case o == "identity":
						mindName = "identity"
						newState, err = identity.HandleEvent(e)
					case o == "merits":
						mindName = "merits"
						newState, err = merits.HandleEvent(e)
					case o == "rockets":
						mindName = "rockets"
						newState, err = rockets.HandleEvent(e)
					case o == "payments":
						mindName = "payments"
						newState, err = payments.HandleEvent(e)
					}
				}
			}

		}
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

func subscribeToKind0Events() (accounts []library.Account) {
	for account, _ := range identity.GetMap() {
		accounts = append(accounts, account)
	}
	return
}
