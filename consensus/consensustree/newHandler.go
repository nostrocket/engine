package consensustree

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/consensus/shares"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

var events = make(map[library.Sha256]nostr.Event)

//HandleConsensusEvent e: the consensus event (kind 640064) to handle
//scEvent: caller should listen on this channel and handle state change event with this ID
//result: caller should send the result after handling event scEvent
//publish: caller should publish (to relays) events received on this channel
func HandleConsensusEvent(e nostr.Event, scEvent chan library.Sha256, scResult chan bool, cPublish chan nostr.Event) error {
	var unmarshalled Kind640064
	err := json.Unmarshal([]byte(e.Content), &unmarshalled)
	if err != nil {
		return err
	}
	if len(unmarshalled.StateChangeEventID) != 64 {
		return fmt.Errorf("invalid state change event ID")
	}
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return handleNewConsensusEvent(unmarshalled, e, scEvent, scResult, cPublish, false)
}

func handleNewConsensusEvent(unmarshalled Kind640064, e nostr.Event, scEvent chan library.Sha256, scResult chan bool, cPublish chan nostr.Event, localEvent bool) error {
	if shares.VotepowerForAccount(e.PubKey) < 1 {
		events[e.ID] = e
		return nil
	}
	startDb()
	if !checkTags(e) {
		events[e.ID] = e
		return nil
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
	_, latestHandledHeight := getLatestHandled()
	if unmarshalled.Height > latestHandledHeight+1 {
		events[e.ID] = e
		return nil
	}
	if unmarshalled.Height < 1 {
		return nil
	}
	for _, event := range current {
		if event.Permille == 1000 {
			return nil //fmt.Errorf("we already have 1000 permille for this height")
		}
		if event.StateChangeEventHandled {
			if unmarshalled.StateChangeEventID != event.StateChangeEventID {
				return fmt.Errorf("we have already handled a different event at this height, cannot process two different events at the same height without wreaking havoc - todo: 4536g45")
				//todo rebuild state if we see a different inner event getting >500 permille at this height. Delete our consensus event if we have produced one for this height and sign the >500 permille one instead.
				//store a checkpoint for the >500 permille state at this height (store to disk) reset and rebuild state, and only validate consensus events with this state change event ID at this height.
			}
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
		return err
	}
	permille, err := shares.Permille(votepower, totalVp)
	if err != nil {
		return err
	}
	currentInner.Permille = permille
	//todo verify current bitcoin height, only upsert if claimed == current
	//currentState.data[unmarshalled.Height][unmarshalled.StateChangeEventID] = currentInner
	//currentState.persistToDisk()
	//todo we are not persisting to disk in live mode
	if currentInner.Permille < 1 {
		return fmt.Errorf("permille is less than 1")
	}

	//
	//continue to copy from old handler. If this is the next height, and inner event has not been handled yet, and >500 permille, then process inner (check the EOSE handler to see what we need to do, take some action if inner event fails)
	//then re-process all events in the events map
	//any time we successfully handle an event, check the map and delete it from there so we don't fuck memory

	//if we have votepower AND have not signed AND this is the next height (AND inner event is valid) THEN sign and broadcast consesnus event

	//what if event is below 500 permille? we still need to sign because otherwise it will never get above 500 permille
	//solution: rebuild state if we see a different inner event getting >500 permille at this height
	//we should always work from checkpointed state if it exists, store local checkpoints every time we pass 500 permille.
	if currentInner.StateChangeEventHeight == latestHandledHeight+1 {
		if localEvent {
			currentInner.StateChangeEventHandled = true
		}
		if !currentInner.StateChangeEventHandled {
			scEvent <- currentInner.StateChangeEventID
			result := <-scResult
			if !result {
				return fmt.Errorf("state change event failed")
			}
			currentInner.StateChangeEventHandled = true
		}
		if shares.VotepowerForAccount(actors.MyWallet().Account) > 0 && !currentInner.IHaveSigned {
			//todo problem: we will sign more than once like this.
			//Solution: go through all previous heights and delete any that we have signed more than once OR don't sign anything until we reach the current bitcoin height or haven't seen any new consensus events for x seconds
			//solution: if this event is already signed by us, and we are now seeing another consensus event (we have signed more than once) delete the newest one - send a kind5 event to delete.
			ce, err := produceConsensusEvent(Kind640064{
				StateChangeEventID: currentInner.StateChangeEventID,
				Height:             currentInner.StateChangeEventHeight,
				BitcoinHeight:      currentInner.BitcoinHeight,
			})
			if err != nil {
				return err
			}
			if err == nil {
				cPublish <- ce
				currentInner.IHaveSigned = true
			}
		}

	}
	current[unmarshalled.StateChangeEventID] = currentInner
	currentState.data[currentInner.StateChangeEventHeight] = current
	return nil
}

func CreateNewConsensusEvent(e nostr.Event, publish chan nostr.Event) error {
	if shares.VotepowerForAccount(actors.MyWallet().Account) < 1 {
		return fmt.Errorf("current wallet has no votepower")
	}
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	_, height := getLatestHandled()
	inner := Kind640064{
		StateChangeEventID: e.ID,
		Height:             height + 1,
		BitcoinHeight:      0, //todo bitcoin height
	}
	newConsensusEvent, err := produceConsensusEvent(inner)
	if err != nil {
		return err
	}
	err = handleNewConsensusEvent(inner, newConsensusEvent, make(chan library.Sha256), make(chan bool), make(chan nostr.Event), true)
	if err != nil {
		return err
	}
	publish <- newConsensusEvent
	return nil
}

func produceConsensusEvent(data Kind640064) (nostr.Event, error) {
	var t = nostr.Tags{}
	t = append(t, nostr.Tag{"e", actors.IgnitionEvent, "", "root"})
	var heighest int64
	var eID library.Sha256
	//find the latest stateChangeEvent that we have signed
	for i, m := range currentState.data {
		for sha256, event := range m {
			if event.IHaveSigned {
				if i >= heighest && !event.IHaveReplaced {
					eID = sha256
					heighest = i
				}
			}
		}
	}
	if len(eID) != 64 {
		t = append(t, nostr.Tag{"e", actors.ConsensusTree, "", "reply"})
	} else {
		t = append(t, nostr.Tag{"e", eID, "", "reply"})
	}

	j, err := json.Marshal(data)
	if err != nil {
		return nostr.Event{}, err
	}
	n := nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Now(),
		Kind:      640064,
		Tags:      t,
		Content:   fmt.Sprintf("%s", j),
	}
	n.ID = n.GetID()
	n.Sign(actors.MyWallet().PrivateKey)
	return n, nil
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
