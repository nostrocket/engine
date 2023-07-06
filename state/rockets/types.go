package rockets

import (
	"nostrocket/engine/library"
)

type Rocket struct {
	RocketID  library.RocketID
	CreatedBy library.Account
	ProblemID library.Sha256
	MissionID library.Sha256
}

type Mapped map[library.RocketID]Rocket

//Kind640600 STATUS:DRAFT
type Kind640600 struct {
	RocketID library.RocketID `json:"rocket_id" status:"draft"`
	Problem  library.Sha256   `json:"problem_id" status:"draft"`
}
