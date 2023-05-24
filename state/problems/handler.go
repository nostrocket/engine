package problems

import (
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/state/identity"
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
		case 641802:
			return handle641802(event)
		}
	}
	return nil, fmt.Errorf("no state changed")
}

func handle641802(event nostr.Event) (m Mapped, e error) {
	var updates int64 = 0
	if anchor, ok := library.GetReply(event); ok {
		if identity.IsUSH(event.PubKey) {
			if currentProblem, problemExists := currentState.data[anchor]; problemExists {
				if currentProblem.CreatedBy == event.PubKey || identity.IsMaintainer(event.PubKey) {
					if len(event.Content) > 0 && event.Content != currentProblem.Body {
						currentProblem.Body = event.Content
						updates++
					}
					if title, ok := library.GetFirstTag(event, "title"); ok {
						if currentProblem.Title != title && len(title) > 0 {
							currentProblem.Title = title
							updates++
						}
					}
					if updates > 0 {
						currentState.upsert(currentProblem.UID, currentProblem)
						return getMap(), nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("no state changed")
}

func handle641800(event nostr.Event) (m Mapped, e error) {
	//fmt.Printf("%#v", event)
	//var updates int64 = 0
	if parent, ok := library.GetReply(event); ok {
		//exception for ignition problem
		if len(currentState.data) == 0 && event.PubKey == actors.IgnitionAccount && parent == actors.StateChangeRequests {
			return insertProblem(event, actors.StateChangeRequests)
		} else {
			if _, exists := currentState.data[event.ID]; !exists {
				if identity.IsUSH(event.PubKey) {
					if _, parentExists := currentState.data[parent]; parentExists {
						return insertProblem(event, parent)
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("no state changed")
}

func insertProblem(event nostr.Event, parent library.Sha256) (m Mapped, e error) {
	p := Problem{
		UID:       event.ID,
		Parent:    parent,
		Title:     event.Content,
		Body:      "",
		Closed:    false,
		ClaimedAt: 0,
		ClaimedBy: "",
		CreatedBy: event.PubKey,
	}
	currentState.upsert(p.UID, p)
	return getMap(), nil
}