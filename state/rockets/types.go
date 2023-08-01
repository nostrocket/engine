package rockets

import (
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/library"
	"nostrocket/state"
)

type RocketData struct {
	data  Mapped
	mutex *deadlock.Mutex
}

type Mapped map[library.RocketID]state.Rocket

//Kind640600 STATUS:DRAFT
type Kind640600 struct {
	RocketID library.RocketID `json:"rocket_id" status:"draft"`
	Problem  library.Sha256   `json:"problem_id" status:"draft"`
}
