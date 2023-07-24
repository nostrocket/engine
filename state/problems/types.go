package problems

import (
	"nostrocket/engine/library"
)

type Problem struct {
	UID       library.Sha256 //this is the ID of the anchor event
	Parent    library.Sha256
	Title     string
	Body      string
	Closed    bool
	ClaimedAt int64 //block height
	ClaimedBy library.Account
	CreatedBy library.Account
	Rocket    library.RocketID
	Tags      []string
}

type Mapped map[library.RocketID]Problem
