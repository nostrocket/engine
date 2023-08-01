package eventconductor

import (
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/state/payments"
	"nostrocket/state/rockets"
)

type CurrentState struct {
	Identity any             `json:"identity"`
	Merits   any             `json:"merits"`
	Replay   any             `json:"replay"`
	Rockets  rockets.Mapped  `json:"rockets"`
	Problems any             `json:"problems"`
	Payments payments.Mapped `json:"payments"`
	mu       *deadlock.Mutex
}

func (c *CurrentState) Wire() (w WireState) {
	w.Payments = c.Payments
	w.Identity = c.Identity
	w.Problems = c.Problems
	w.Replay = c.Replay
	w.Merits = c.Merits
	w.Rockets = make(MappedRockets)
	for id, r := range c.Rockets {
		newRocket := Rocket{
			RocketUID:  r.RocketUID,
			RocketName: r.RocketName,
			CreatedBy:  r.CreatedBy,
			ProblemID:  r.ProblemID,
			MissionID:  r.MissionID,
			Products:   []payments.Product{},
		}
		if d, exists := c.Payments.Products[id]; exists {
			for _, product := range d {
				newRocket.Products = append(newRocket.Products, product)
			}
		}
		w.Rockets[id] = newRocket
	}
	return
}

type WireState struct {
	Identity any             `json:"identity"`
	Merits   any             `json:"merits"`
	Replay   any             `json:"replay"`
	Rockets  MappedRockets   `json:"rockets"`
	Problems any             `json:"problems"`
	Payments payments.Mapped `json:"payments"`
}
type MappedRockets map[library.RocketID]Rocket

type Rocket struct {
	RocketUID  library.Sha256
	RocketName string
	CreatedBy  library.Account
	ProblemID  library.Sha256
	MissionID  library.Sha256
	Products   []payments.Product
	Payments   []payments.PaymentRequest
}

var currentState = CurrentState{
	Identity: nil,
	Merits:   nil,
	Replay:   nil,
	Problems: nil,
	Payments: payments.Mapped{},
	mu:       &deadlock.Mutex{},
}

func AppendState(name string, state any) (WireState, bool) {
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
		currentState.Rockets = state.(rockets.Mapped)
	case "problems":
		currentState.Problems = state
	case "payments":
		currentState.Payments = state.(payments.Mapped)
	default:
		return WireState{}, false
	}
	return currentState.Wire(), true
}

func CurrentStateEventBuilder(state string) nostr.Event {
	e := nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Now(),
		Kind:      10311,
		Tags:      nostr.Tags{nostr.Tag{"e", actors.CurrentStates, "", "reply"}, nostr.Tag{"e", actors.IgnitionEvent, "", "root"}},
		Content:   state,
	}
	e.ID = e.GetID()
	e.Sign(actors.MyWallet().PrivateKey)
	return e
}

func GetCurrentStateMap() CurrentState {
	currentState.mu.Lock()
	defer currentState.mu.Unlock()
	return currentState
}
