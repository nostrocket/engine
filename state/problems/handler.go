package problems

import (
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func HandleEvent(event nostr.Event) (m Mapped, e error) {
	startDb()
	if sig, _ := event.CheckSignature(); !sig {
		return
	}
	if event.Kind >= 641800 && event.Kind <= 641899 {
		currentState.mutex.Lock()
		defer currentState.mutex.Unlock()
		switch event.Kind {
		case 641800:
			return handle641800(event)
		}
	}
	return nil, fmt.Errorf("no state changed")
}

func handle641800(event nostr.Event) (m Mapped, e error) {
	//var updates int64 = 0
	if t, ok := library.GetReply(event); ok {
		//exception for ignition problem
		if len(currentState.data) == 0 && event.PubKey == actors.IgnitionAccount && t == actors.StateChangeRequests {
			p := Problem{
				UID:       event.ID,
				Parent:    actors.StateChangeRequests,
				Title:     event.Content,
				Body:      "",
				Closed:    false,
				ClaimedAt: 0,
				ClaimedBy: "",
				CreatedBy: event.PubKey,
			}
			currentState.upsert(p.UID, p)
			return getMap(), nil
		} else {

		}
	}
	return nil, fmt.Errorf("no state changed")
}
