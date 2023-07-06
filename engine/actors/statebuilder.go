package actors

import (
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
)

type CurrentState struct {
	Identity any `json:"identity"`
	Merits   any `json:"merits"`
	Replay   any `json:"replay"`
	Rockets  any `json:"rockets"`
	Problems any `json:"problems"`
	mu       *deadlock.Mutex
}

var currentState = CurrentState{
	Identity: nil,
	Merits:   nil,
	Replay:   nil,
	Problems: nil,
	mu:       &deadlock.Mutex{},
}

func AppendState(name string, state any) (CurrentState, bool) {
	currentState.mu.Lock()
	defer currentState.mu.Unlock()
	switch name {
	case "identity":
		currentState.Identity = state
	case "merits":
		currentState.Merits = state
	case "replay":
		currentState.Replay = state
	case "rockets":
		currentState.Rockets = state
	case "problems":
		currentState.Problems = state
	default:
		return CurrentState{}, false
	}
	return currentState, true
}

func CurrentStateEventBuilder(state string) nostr.Event {
	e := nostr.Event{
		PubKey:    MyWallet().Account,
		CreatedAt: time.Now(),
		Kind:      10311,
		Tags:      nostr.Tags{nostr.Tag{"e", CurrentStates, "", "reply"}, nostr.Tag{"e", IgnitionEvent, "", "root"}},
		Content:   state,
	}
	e.ID = e.GetID()
	e.Sign(MyWallet().PrivateKey)
	return e
}

func CurrentStateMap() CurrentState {
	currentState.mu.Lock()
	defer currentState.mu.Unlock()
	return currentState
}
