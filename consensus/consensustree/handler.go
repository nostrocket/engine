package consensustree

import (
	"encoding/json"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/consensus/shares"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func HandleBatchAfterEOSE(m map[library.Sha256]nostr.Event, wg *deadlock.WaitGroup, eventsToHandle chan library.Sha256) {
	//for each height, we find the inner event with the highest votepower and follow that, producing our own consensus event if we have votepower.

}

//handler
//func (s *db) upsert(key int64, val TreeEvent) {
//	if d, ok := s.data[val.StateChangeEventHeight]; !ok {
//		d = make(map[library.Sha256]TreeEvent)
//		s.data[val.StateChangeEventHeight] = d
//	}
//
//	if d, ok := s.data[val.StateChangeEventHeight][val.StateChangeEventID];
//	d, _ := s.data[val.StateChangeEventHeight]
//	d[val.StateChangeEventID] = val
//
//	for _, event := range s.data[val.Height] {
//		if event.Signer == val.Signer && event.StateChangeEventID == val.StateChangeEventID {
//			return
//		}
//	}
//	s.data[key] = append(s.data[key], val)
//}

func HandleEvent(e nostr.Event) error {
	//todo return the event ID that we should process into state. This can be ignored if it's one that we just produced locally.
	if shares.VotepowerForAccount(e.PubKey) < 1 {
		return fmt.Errorf("%s has no votepower", e.PubKey)
	}
	startDb()
	if !checkTags(e) {
		return fmt.Errorf("%s is not replying to the current consensustree tip", e.ID)
	}
	var unmarshalled Kind640064
	err := json.Unmarshal([]byte(e.Content), &unmarshalled)
	if err != nil {
		library.LogCLI(err.Error(), 3)
		return err
	}
	var current map[library.Sha256]TreeEvent
	var exists bool
	current, exists = currentState.data[unmarshalled.Height]
	if !exists {
		current = make(map[library.Sha256]TreeEvent)
		currentState.data[unmarshalled.Height] = current
	}
	currentInner, cIexists := current[unmarshalled.StateChangeEventID]
	if !cIexists {
		currentInner = TreeEvent{
			StateChangeEventHeight: unmarshalled.Height,
			StateChangeEventID:     unmarshalled.StateChangeEventID,
			Signers:                make(map[library.Account]int64),
			ConsensusEvents:        make(map[library.Sha256]nostr.Event),
			IHaveSigned:            false,
			IHaveReplaced:          false,
			Votepower:              0,
			TotalVotepoweAtHeight:  0,
			Permille:               0,
			BitcoinHeight:          0,
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
	//todo get total current global votepower
	//todo if >500 permille, return the statechangeeventID so that we can process that event, and update our current state to the new height
	//todo verify current bitcoin height, only upsert if claimed == current
	fmt.Println(currentState.data[unmarshalled.Height])
	currentState.data[unmarshalled.Height][unmarshalled.StateChangeEventID] = currentInner
	currentState.persistToDisk()
	return nil
}

func checkTags(e nostr.Event) bool {
	hash, _ := getMyLastest()
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
