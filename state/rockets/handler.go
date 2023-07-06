package rockets

import (
	"encoding/json"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
	"nostrocket/state/identity"
)

func HandleEvent(event nostr.Event) (m Mapped, e error) {
	startDb()
	if !identity.IsUSH(event.PubKey) {
		return nil, fmt.Errorf("event %s: pubkey %s not in identity tree", event.ID, event.PubKey)
	}
	if event.Kind >= 640600 && event.Kind <= 640699 {
		currentState.mutex.Lock()
		defer currentState.mutex.Unlock()
		switch event.Kind {
		case 640600:
			return handle640600(event)
		}
	}
	return m, fmt.Errorf("event %s did not cause a state change", event.ID)
}

func handle640600(event nostr.Event) (m Mapped, err error) {
	var unmarshalled Kind640600
	if err = json.Unmarshal([]byte(event.Content), &unmarshalled); err != nil {
		return m, fmt.Errorf("%s reported for event %s", err.Error(), event.ID)
	}
	if nameTaken(unmarshalled.RocketID) {
		return m, fmt.Errorf("event %s requests creation of new rocket \"%s\" but this name is already taken", event.ID, unmarshalled.RocketID)
	}
	currentState.upsert(unmarshalled.RocketID, Rocket{
		RocketID:  unmarshalled.RocketID,
		CreatedBy: event.PubKey,
		ProblemID: unmarshalled.Problem,
	})
	//currentState.persistToDisk()
	return getMap(), nil
}

func nameTaken(name library.RocketID) bool {
	if _, exists := currentState.data[name]; exists {
		return true
	}
	return false
}
