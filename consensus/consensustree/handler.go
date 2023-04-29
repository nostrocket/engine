package consensustree

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/consensus/shares"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

var handled = make(map[library.Sha256]struct{})

func HandleBatchAfterEOSE(m []nostr.Event, eventsToHandle chan library.Sha256, consensusEventsToPublish chan nostr.Event, innerEventHandlerResult chan bool) {
	//todo if there's more than one fork, simulate all of them to find the longest chain, and delete all our own consensus events from all other chains.
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	//for each height, we find the inner event with the highest votepower and follow that, producing our own consensus event if we have votepower.
	//if event is last one at height, return inner event id on channel. Then wait on waitForCaller before processing next one.
	var eventsGroupedByHeight [][]nostr.Event
	var sorted []nostr.Event
	for _, event := range m {
		var unmarshalled Kind640064
		err := json.Unmarshal([]byte(event.Content), &unmarshalled)
		if err != nil {
			library.LogCLI("event "+event.ID+": "+err.Error(), 3)
		} else {
			sorted = append(sorted, event)
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		var unmarshalledi Kind640064
		var unmarshalledj Kind640064
		json.Unmarshal([]byte(sorted[i].Content), &unmarshalledi)
		json.Unmarshal([]byte(sorted[j].Content), &unmarshalledj)
		if unmarshalledi.Height > unmarshalledj.Height {
			return false
		}
		return true
	})
	var currentHeight int64
	var newEvents []nostr.Event
	for i, event := range sorted {
		var unmarshalled Kind640064
		json.Unmarshal([]byte(event.Content), &unmarshalled)
		if currentHeight == unmarshalled.Height {
			newEvents = append(newEvents, event)
		}
		if currentHeight+1 == unmarshalled.Height || i == len(sorted)-1 {
			eventsGroupedByHeight = append(eventsGroupedByHeight, newEvents)
			newEvents = []nostr.Event{event}
			currentHeight++
		}
	}
	for _, event := range eventsGroupedByHeight {
		var innerEventToReturn library.Sha256
		var treeEvent TreeEvent
		for _, n := range event {
			var unmarshalled Kind640064
			err := json.Unmarshal([]byte(n.Content), &unmarshalled)

			if err == nil {
				if _, exists := handled[unmarshalled.StateChangeEventID]; !exists {
					tree, innerEvent, err := handleConsensusEvent(n)
					if err != nil {
						//fmt.Printf("\n61\n%#v\n", n)
						//library.LogCLI(err.Error(), 2)
						continue
					} else {
						innerEventToReturn = innerEvent
						treeEvent = tree
					}
				}
			}
		}
		//if permille > 500 we handle the inner event at the end of each height
		//if we have votepower, we handle the inner event as well, so that we can broadcast our signed consensus event
		//if no votepower and not >500 permille, we stop at this height and return.
		//todo if we have votepower we handle the inner event and need to return regardless and then handle inner event which has the greatest votepower at this height
		if (treeEvent.Permille > 500 || shares.VotepowerForAccount(actors.MyWallet().Account) > 0) && len(innerEventToReturn) == 64 {
			eventsToHandle <- innerEventToReturn
			result := <-innerEventHandlerResult
			if result {
				handled[treeEvent.StateChangeEventID] = struct{}{}
				//put consensus event into state
				existing, exists := currentState.data[treeEvent.StateChangeEventHeight]
				if !exists {
					existing = make(map[library.Sha256]TreeEvent)
				}
				treeEvent.StateChangeEventHandled = true
				existing[treeEvent.StateChangeEventID] = treeEvent
				currentState.data[treeEvent.StateChangeEventHeight] = existing
				if shares.VotepowerForAccount(actors.MyWallet().Account) > 0 && !treeEvent.IHaveSigned {
					//todo validate that we are actually updating treeEvent.IHaveSigned
					e, err := produceEvent(treeEvent.StateChangeEventID, 0)
					if err != nil {
						library.LogCLI(err.Error(), 1)
					}
					if err == nil {
						consensusEventsToPublish <- e
					}
				}
				currentState.persistToDisk()
			}
			if !result {
				//todo delete the consensus event if its ours
				//do not put conesnsus event into state
				fmt.Println(113, innerEventToReturn, "failed")
				continue
			}
		}
	}
}

func HandleEvent(n nostr.Event) error {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	tree, _, err := handleConsensusEvent(n)
	if err != nil {
		return err
	}
	existing, exists := currentState.data[tree.StateChangeEventHeight]
	if !exists {
		existing = make(map[library.Sha256]TreeEvent)
	}
	tree.StateChangeEventHandled = true
	existing[tree.StateChangeEventID] = tree
	currentState.data[tree.StateChangeEventHeight] = existing
	currentState.persistToDisk()
	return nil
}

func handleConsensusEvent(e nostr.Event) (t TreeEvent, l library.Sha256, er error) {
	//todo if we are on the wrong side of a fork (lowest votepower) set IHaveReplaced to true
	//we can't check for our current latest height that we have signed becuase there might be multiples if we changed fork
	if shares.VotepowerForAccount(e.PubKey) < 1 {
		return t, l, fmt.Errorf("%s has no votepower", e.PubKey)
	}
	startDb()
	if !checkTags(e) {
		return t, l, fmt.Errorf("%s is not replying to the current consensustree tip", e.ID)
	}
	var unmarshalled Kind640064
	err := json.Unmarshal([]byte(e.Content), &unmarshalled)
	if err != nil {
		library.LogCLI(err.Error(), 3)
		return t, l, err
	}

	var current map[library.Sha256]TreeEvent
	var exists bool
	current, exists = currentState.data[unmarshalled.Height]
	if !exists {
		current = make(map[library.Sha256]TreeEvent)
		//currentState.data[unmarshalled.Height] = current
	}
	currentInner, cIexists := current[unmarshalled.StateChangeEventID]
	if !cIexists {
		currentInner = TreeEvent{
			StateChangeEventHeight:  unmarshalled.Height,
			StateChangeEventID:      unmarshalled.StateChangeEventID,
			StateChangeEventHandled: false,
			Signers:                 make(map[library.Account]int64),
			ConsensusEvents:         make(map[library.Sha256]nostr.Event),
			IHaveSigned:             false,
			IHaveReplaced:           false,
			Votepower:               0,
			TotalVotepoweAtHeight:   0,
			Permille:                0,
			BitcoinHeight:           0,
		}
	}
	//validate:
	//if this event == current height &&
	//if current height permille < 500 (could be new fork or could be same, doesn't matter) || innerEvent == this (means that we are just adding votepower)
	latestHandledEvent, latestHandledHeight := getLatestHandled()
	if unmarshalled.Height != latestHandledHeight && unmarshalled.Height != latestHandledHeight+1 {
		return t, l, fmt.Errorf("invalid height on consensus event")
	}
	for _, event := range current {
		if event.Permille == 1000 {
			return t, l, fmt.Errorf("we already have 1000 permille for this height")
		}
	}
	if unmarshalled.Height == latestHandledHeight {
		if unmarshalled.StateChangeEventID != latestHandledEvent {
			return t, l, fmt.Errorf("we have already handled a different event at this height, cannot process two different events at the same height without wreaking havoc, make reset if you need to follow a different fork")
		}
	}

	currentInner.Signers[e.PubKey] = shares.VotepowerForAccount(e.PubKey)
	currentInner.ConsensusEvents[e.ID] = e
	if e.PubKey == actors.MyWallet().Account {
		currentInner.IHaveSigned = true
	}
	var votepower int64
	for account, _ := range currentInner.Signers {
		votepower = votepower + shares.VotepowerForAccount(account)
	}
	totalVp, err := shares.TotalVotepower()
	if err != nil {
		return t, l, err
	}
	permille, err := shares.Permille(votepower, totalVp)
	if err != nil {
		return t, l, err
	}
	currentInner.Permille = permille
	//todo verify current bitcoin height, only upsert if claimed == current
	//currentState.data[unmarshalled.Height][unmarshalled.StateChangeEventID] = currentInner
	//currentState.persistToDisk()
	//todo we ae not persisting to disk in live mode
	if currentInner.Permille < 1 {
		return t, l, fmt.Errorf("permille is less than 1")
	}
	if currentInner.Permille > 0 {
		return currentInner, currentInner.StateChangeEventID, nil
	}
	return t, l, fmt.Errorf("no inner")
}

func checkTags(e nostr.Event) bool {
	hash, _ := getLatestHandled()
	for _, tag := range e.Tags {
		if len(tag) == 4 {
			if tag[0] == "e" {
				if tag[1] == hash {
					return true
				}
			}
		}
	}
	return false
}

//when we are replaying, we handle the consensus event BEFORE replaying the state change event, then we update the consensus data in case votepower changed. The handler should reply with the accounts that have signed this state change event hash so that the conductor can check the permille.
//the conductor should only replay the state change event after it sees the next consensus height (or end of list), this is to prevent playing state change messages until we know that there are no alternative consensus branches.

//when we are in live mode, we play the state change event, BEFORE producing the consensus event.

//fetch ALL events in the nostrocket tree, map them all.
//Once we get the EOSE, process each Kind 640064 event in order from the ignition event, find the height of each and put them into a map then handle them all, replaying state change events too.
//buffer post-EOSE incoming events in a queue which we pick up after processing everything recieved before EOSE
//don't store any state locally, replay everything from the start each time
//don't publish state until we fully sync
//

//make state changes visible in terminal logs, show which mind state was updated
