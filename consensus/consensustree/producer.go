package consensustree

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

var lock = &deadlock.Mutex{}

func ProduceEvent(stateChangeEventID library.Sha256, bitcoinHeight int64) (nostr.Event, error) {
	lock.Lock()
	defer lock.Unlock()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	if len(currentState.data) == 0 && actors.MyWallet().Account == actors.IgnitionAccount {
		//exception for first event
		return produceEvent(stateChangeEventID, bitcoinHeight, 1, nostr.Tags{nostr.Tag{"e", actors.ConsensusTree, "", "reply"}})
	}
	var heighest int64
	var eventID library.Sha256
	//find the latest stateChangeEvent that we have signed
	for i, m := range currentState.data {
		for sha256, event := range m {
			if event.IHaveSigned {
				if i >= heighest && !event.IHaveReplaced {
					eventID = sha256
					heighest = i
				}
			}
		}
	}
	if heighest > 0 && len(eventID) == 64 {
		return produceEvent(stateChangeEventID, bitcoinHeight, heighest+1, nostr.Tags{
			nostr.Tag{"e", eventID, "", "reply"},
		})
	}
	//todo problem: newly created votepower doesn't produce consensenustree events
	return nostr.Event{}, fmt.Errorf("not implemented")
}

func produceEvent(stateChangeEventID library.Sha256, bitcoinHeight int64, eventHeight int64, tags nostr.Tags) (nostr.Event, error) {
	var t = nostr.Tags{}
	t = append(t, nostr.Tag{"e", actors.IgnitionEvent, "", "root"})
	t = append(t, tags...)
	j, err := json.Marshal(Kind640064{
		StateChangeEventID: stateChangeEventID,
		Height:             eventHeight,
		BitcoinHeight:      bitcoinHeight,
	})
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
