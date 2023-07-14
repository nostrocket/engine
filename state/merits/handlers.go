package merits

import (
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
	"nostrocket/state/identity"
	"nostrocket/state/rockets"
)

func HandleEvent(event nostr.Event) (m Mapped, err error) {
	startDb()
	if !identity.IsUSH(event.PubKey) {
		return nil, fmt.Errorf("event %s: pubkey %s not in identity tree", event.ID, event.PubKey)
	}
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	switch event.Kind {
	case 1:
		return handleByTags(event)
	default:
		return nil, fmt.Errorf("I am the merits mind, event %s was sent to me but I don't know how to handle kind %d", event.ID, event.Kind)
	}
}

func handleByTags(event nostr.Event) (m Mapped, e error) {
	if operation, ok := library.GetFirstTag(event, "op"); ok {
		ops := strings.Split(operation, ".")
		if len(ops) > 2 {
			if ops[1] == "merits" {
				switch o := ops[2]; {
				case o == "register":
					return handleRegistration(event)
				}
			}
		}
	}
	return nil, fmt.Errorf("no valid operation found 35645ft")
}

func handleRegistration(event nostr.Event) (m Mapped, e error) {
	var rocketID string
	var founder library.Account
	var ok bool
	if rocketID, ok = library.GetOpData(event); !ok {
		return nil, fmt.Errorf("no valid operation found 678yug")
	}
	if founder, ok = rockets.RocketCreators()[rocketID]; !ok {
		return m, fmt.Errorf("%s tried to create a new cap table for rocket %s, but the rocket mind reports no such rocket exists", event.ID, rocketID)
	}
	if founder != event.PubKey {
		return m, fmt.Errorf("%s tried to create a new cap table for rocket %s, but the rocket is owned by %s", event.ID, rocketID, founder)
	}
	if err := makeNewCapTable(rocketID); err != nil {
		return m, fmt.Errorf("%s tried to create a new cap table for rocket %s, but %s", event.ID, rocketID, err.Error())
	}
	d := currentState[rocketID]
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.data[event.PubKey] = Merit{
		RocketID:               rocketID,
		LeadTimeLockedMerits:   1,
		LeadTime:               1,
		LastLtChange:           0, //todo current bitcoin height
		LeadTimeUnlockedMerits: 0,
	}
	currentState[rocketID] = d
	return getMapped(), nil

}
