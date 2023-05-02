package consensustree

//todo are we handling out-of order consensus events? make sure
import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/consensus/shares"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

var events = make(map[library.Sha256]nostr.Event)
var num int64 = 0
var debug = false

//HandleConsensusEvent e: the consensus event (kind 640064) to handle
//scEvent: caller should listen on this channel and handle state change event with this ID
//result: caller should send the result after handling event scEvent
//publish: caller should publish (to relays) events received on this channel
func HandleConsensusEvent(e nostr.Event, scEvent chan library.Sha256, scResult chan bool, cPublish chan nostr.Event) error {
	if debug {
		cPublish <- deleteEvent(e.ID)
		num++
		println(num)
		return nil
	}
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
	if c, ok := getCheckpoint(unmarshalled.Height); ok {
		if c.StateChangeEventID != unmarshalled.StateChangeEventID {
			if e.PubKey == actors.MyWallet().Account {
				cPublish <- deleteEvent(e.ID)
			}
			return fmt.Errorf("we have a checkpoint at this height and it doesn't match the event provided")
		}
	}
	//if e.PubKey == actors.MyWallet().Account {
	//	fmt.Println(59)
	//}
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
	if e.PubKey == actors.MyWallet().Account && currentInner.StateChangeEventHandled && currentInner.IHaveSigned {
		cPublish <- deleteEvent(e.ID)
		return nil
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
	//todo we are not persisting to disk in live mode
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
		if shares.VotepowerForAccount(actors.MyWallet().Account) > 0 && !currentInner.IHaveSigned {
			ce, err := produceConsensusEvent(Kind640064{
				StateChangeEventID: currentInner.StateChangeEventID,
				Height:             currentInner.StateChangeEventHeight,
				BitcoinHeight:      currentInner.BitcoinHeight,
			})
			if err != nil {
				library.LogCLI(err.Error(), 1)
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
				r = append(r, deleteEvent(ourConsensusEvents[k-1].ID))
			}
		}
	}
	return
}

func deleteEvent(id library.Sha256) (r nostr.Event) {
	r = nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Now(),
		Kind:      5,
		Tags: nostr.Tags{nostr.Tag{
			"e", id},
		},
		Content: "woops",
	}
	r.ID = r.GetID()
	r.Sign(actors.MyWallet().PrivateKey)
	return
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
