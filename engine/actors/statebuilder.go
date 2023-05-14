package actors

import (
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
)

type CurrentState struct {
	Identity any `json:"identity"`
	Shares   any `json:"shares"`
	Replay   any `json:"replay"`
	Mirvs    any `json:"mirvs"`
	Problems any `json:"problems"`
	mu       *deadlock.Mutex
}

var currentState = CurrentState{
	Identity: nil,
	Shares:   nil,
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
	case "shares":
		currentState.Shares = state
	case "replay":
		currentState.Replay = state
	case "mirvs":
		currentState.Mirvs = state
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
		Kind:      10310,
		Tags:      nostr.Tags{nostr.Tag{"e", CurrentStates, "", "reply"}, nostr.Tag{"e", IgnitionEvent, "", "root"}},
		Content:   state,
	}
	e.ID = e.GetID()
	e.Sign(MyWallet().PrivateKey)
	return e
}
