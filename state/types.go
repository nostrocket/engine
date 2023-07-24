package state

import (
	"nostrocket/engine/library"
)

type Rocket struct {
	RocketUID  library.Sha256
	RocketName string
	CreatedBy  library.Account
	ProblemID  library.Sha256
	MissionID  library.Sha256
}

var rockets = make(map[library.Sha256]Rocket)

func (r Rocket) upsert() any {
	rockets[r.RocketUID] = r
	return rockets
}
