package shares

import (
	"encoding/json"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
	"nostrocket/state/identity"
	"nostrocket/state/mirv"
)

//create new mirv shares
//take in the name of the rocket, and give the creator 1 share with 1 lead time
//do this for nostrocket itself too
//create rocket name first. Then another event to create first share.

func HandleEvent(event nostr.Event) (m Mapped, err error) {
	startDb()
	if !identity.IsUSH(event.PubKey) {
		return nil, fmt.Errorf("event %s: pubkey %s not in identity tree", event.ID, event.PubKey)
	}
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	switch event.Kind {
	case 640208:
		//Create New Mirv Cap Table
		return handle640208(event)
	default:
		return nil, fmt.Errorf("I am the shares mind, event %s was sent to me but I don't know how to handle kind %d", event.ID, event.Kind)
	}
}

func handle640208(event nostr.Event) (m Mapped, err error) {
	var unmarshalled Kind640208
	if err = json.Unmarshal([]byte(event.Content), &unmarshalled); err != nil {
		return m, fmt.Errorf("%s reported for event %s", err.Error(), event.ID)
	}
	var founder library.Account
	var ok bool
	if founder, ok = mirv.Names()[unmarshalled.RocketID]; !ok {
		return m, fmt.Errorf("%s tried to create a new cap table for mirv %s, but the mirv mind reports no such mirv exists", event.ID, unmarshalled.RocketID)
	}
	if founder != event.PubKey {
		return m, fmt.Errorf("%s tried to create a new cap table for mirv %s, but the mirv is owned by %s", event.ID, unmarshalled.RocketID, founder)
	}
	if err = makeNewCapTable(unmarshalled.RocketID); err != nil {
		return m, fmt.Errorf("%s tried to create a new cap table for mirv %s, but %s", event.ID, unmarshalled.RocketID, err.Error())
	}
	d := currentState[unmarshalled.RocketID]
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.data[event.PubKey] = Share{
		LeadTimeLockedShares:   1,
		LeadTime:               1,
		LastLtChange:           0, //todo current bitcoin height
		LeadTimeUnlockedShares: 0,
	}
	currentState[unmarshalled.RocketID] = d
	//d.persistToDisk()
	return getMapped(), nil
}
