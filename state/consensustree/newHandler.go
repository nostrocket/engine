package consensustree

//todo are we handling out-of order consensus events? make sure
//todo: what about if we are new votepower and haven't signed anything yet?
import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/actors"
	"nostrocket/engine/helpers"
	"nostrocket/engine/library"
	"nostrocket/state/merits"
)

var events = make(map[library.Sha256]nostr.Event)
var num int64 = 0
var debug = false

//HandleConsensusEvent e: the consensus event (kind 640001) to handle
//scEvent: caller should listen on this channel and handle state change event with this ID
//result: caller should send the result after handling event scEvent
//publish: caller should publish (to relays) events received on this channel
func HandleConsensusEvent(e nostr.Event, scEvent chan library.Sha256, scResult chan bool, cPublish chan nostr.Event, localEvent bool) error {
	if debug {
		cPublish <- helpers.DeleteEvent(e.ID, "woops")
		num++
		println(num)
		return nil
	}
	var unmarshalled Kind640001
	err := json.Unmarshal([]byte(e.Content), &unmarshalled)
	if err != nil {
		return err
	}
	if len(unmarshalled.StateChangeEventID) != 64 {
		return fmt.Errorf("invalid state change event ID")
	}
	if unmarshalled.StateChangeEventID == "519eb09f82997cdb8ffcb3529b542392eba9500265c484ec1441843c740648bd" {
		cPublish <- helpers.DeleteEvent(e.ID, "invalid state change event")
		return nil
	}
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return handleNewConsensusEvent(unmarshalled, e, scEvent, scResult, cPublish, localEvent)
}

func handleNewConsensusEvent(unmarshalled Kind640001, e nostr.Event, scEvent chan library.Sha256, scResult chan bool, cPublish chan nostr.Event, localEvent bool) error {
	if merits.VotepowerForAccount(e.PubKey) < 1 {
		events[e.ID] = e
		return nil
	}
	startDb()
	if c, ok := getCheckpoint(unmarshalled.Height); ok {
		if c.StateChangeEventID != unmarshalled.StateChangeEventID {
			if e.PubKey == actors.MyWallet().Account {
				cPublish <- helpers.DeleteEvent(e.ID, "invalid checkpoint detected")
				actors.LogCLI(fmt.Sprintf("attempting to delete consensus event created %f seconds ago", time.Since(e.CreatedAt).Seconds()), 2)
			}
			return fmt.Errorf("trying to parse %s at height %d, but we already have a checkpoint for %s at height %d", unmarshalled.StateChangeEventID, unmarshalled.Height, c.StateChangeEventID, c.StateChangeEventHeight)
		}
	}
	var current map[library.Sha256]TreeEvent
	var exists bool
	current, exists = currentState.data[unmarshalled.Height]
	if !exists {
		current = make(map[library.Sha256]TreeEvent)
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
	if e.PubKey == actors.MyWallet().Account && currentInner.StateChangeEventHandled && currentInner.IHaveSigned { //&& !localEvent
		timestamp := time.Now().Unix()
		var del []library.Sha256
		for _, event := range currentInner.ConsensusEvents {
			if event.PubKey == actors.MyWallet().Account {
				if event.CreatedAt.Unix() <= timestamp {
					timestamp = event.CreatedAt.Unix()
				}
			}
		}
		for _, event := range currentInner.ConsensusEvents {
			if event.PubKey == actors.MyWallet().Account {
				if event.CreatedAt.Unix() > timestamp {
					del = append(del, event.ID)

				}
			}
		}
		var err error = nil
		for _, sha256 := range del {
			cPublish <- helpers.DeleteEvent(sha256, "duplicate consensus event")
			if sha256 == e.ID {
				err = fmt.Errorf("surplus consensus event detected")
			}
		}
		if err != nil {
			return err
		}
	}

	if !checkTags(e) && !currentInner.StateChangeEventHandled {
		events[e.ID] = e
		return nil
	}
	_, latestHandledHeight := getLatestHandled()
	if unmarshalled.Height > latestHandledHeight+1 && !currentInner.StateChangeEventHandled {
		events[e.ID] = e
		return nil
	}
	if unmarshalled.Height < 1 {
		return nil
	}
	var aStateChangeEventHasAlreadyBeenHandledAtThisHeight bool
	for _, event := range current {
		if event.Permille == 1000 {
			return nil //fmt.Errorf("we already have 1000 permille for this height")
		}
		if event.StateChangeEventHandled {
			if unmarshalled.StateChangeEventID != event.StateChangeEventID {
				aStateChangeEventHasAlreadyBeenHandledAtThisHeight = true
			}
		}
	}
	currentInner.Signers[e.PubKey] = merits.VotepowerForAccount(e.PubKey)
	currentInner.ConsensusEvents[e.ID] = e
	if e.PubKey == actors.MyWallet().Account {
		currentInner.IHaveSigned = true
	}
	var votepower int64
	for account, _ := range currentInner.Signers {
		votepower = votepower + merits.VotepowerForAccount(account)
	}
	totalVp, err := merits.TotalVotepower()
	if err != nil {
		return err
	}
	permille, err := merits.Permille(votepower, totalVp)
	if err != nil {
		return err
	}
	currentInner.Permille = permille
	//todo verify current bitcoin height, only upsert if claimed == current
	if currentInner.Permille < 1 {
		return fmt.Errorf("permille is less than 1")
	}
	if currentInner.StateChangeEventHeight == latestHandledHeight+1 {
		if localEvent {
			currentInner.StateChangeEventHandled = true
		}
		if !currentInner.StateChangeEventHandled && !aStateChangeEventHasAlreadyBeenHandledAtThisHeight {
			scEvent <- currentInner.StateChangeEventID
			result := <-scResult
			if !result {
				return fmt.Errorf("state change event failed")
			}
			currentInner.StateChangeEventHandled = true
		}
		if merits.VotepowerForAccount(actors.MyWallet().Account) > 0 && !currentInner.IHaveSigned {
			ce, err := produceConsensusEvent(Kind640001{
				StateChangeEventID: currentInner.StateChangeEventID,
				Height:             currentInner.StateChangeEventHeight,
				BitcoinHeight:      currentInner.BitcoinHeight,
			})
			if err != nil {
				actors.LogCLI(err.Error(), 1)
				return err
			}
			if err == nil {
				cPublish <- ce
				//currentInner.IHaveSigned = true
			}
		}
	}
	current[unmarshalled.StateChangeEventID] = currentInner
	currentState.data[currentInner.StateChangeEventHeight] = current
	if currentInner.Permille > 500 {
		setCheckpoint(Checkpoint{
			StateChangeEventHeight: currentInner.StateChangeEventHeight,
			StateChangeEventID:     currentInner.StateChangeEventID,
			BitcoinHeight:          0, //todo bitcoin height
			CreatedAt:              time.Now().Unix(),
		})
	}
	//todo check if we have any conflicting consensus states where we have handled the state change event but something else at the same height has a greater permille, then reset and follow that one instead. Store it only as a temporary checkpoint if <500 permille.
	//todo rebuild state if we see a different inner event at this height with a > permille than current. Delete our consensus event if we have produced one for this height and sign the > permille one instead.
	return nil
}

func DeleteDuplicateConsensusEvents() []nostr.Event {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return deleteDuplicateConsensusEvents()
}

func deleteDuplicateConsensusEvents() (r []nostr.Event) {
	for _, m := range currentState.data {
		for _, event := range m {
			var ourConsensusEvents []nostr.Event
			for _, n := range event.ConsensusEvents {
				if n.PubKey == actors.MyWallet().Account {
					ourConsensusEvents = append(ourConsensusEvents, n)
				}
			}
			sort.Slice(ourConsensusEvents, func(i, j int) bool {
				return ourConsensusEvents[i].CreatedAt.Unix() < ourConsensusEvents[j].CreatedAt.Unix()
			})
			for k := len(ourConsensusEvents); k > 1; k-- {
				r = append(r, helpers.DeleteEvent(ourConsensusEvents[k-1].ID, "duplicatae consensus event"))
			}
		}
	}
	return
}

func CreateNewConsensusEvent(ev nostr.Event) (n nostr.Event, e error) {
	if merits.VotepowerForAccount(actors.MyWallet().Account) < 1 {
		return n, fmt.Errorf("current wallet has no votepower")
	}
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	_, height := getLatestHandled()
	inner := Kind640001{
		StateChangeEventID: ev.ID,
		Height:             height + 1,
		BitcoinHeight:      0, //todo bitcoin height
	}
	newConsensusEvent, err := produceConsensusEvent(inner)
	if err != nil {
		return n, err
	}
	//err = handleNewConsensusEvent(inner, newConsensusEvent, make(chan library.Sha256), make(chan bool), make(chan nostr.Event), true)
	//if err != nil {
	//	return err
	//}
	//publish <- newConsensusEvent
	return newConsensusEvent, e
}

func produceConsensusEvent(data Kind640001) (nostr.Event, error) {
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
		Kind:      640001,
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
