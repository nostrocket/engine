package merits

import (
	"nostrocket/engine/library"
)

type Merit struct {
	LeadTimeLockedMerits   int64
	LeadTime               int64 // in multiples of 2016 blocks (2 weeks)
	LastLtChange           int64 // btc height
	LeadTimeUnlockedMerits int64 //Approved Expenses can be swept into here and sold even if Participant's LT is >0
	OpReturnAddresses      []string
}

//Kind640200 STATUS:DRAFT
//Used for adjusting lead time
type Kind640200 struct {
	AdjustLeadTime string //+ or -
	LockMerits     int64
	UnlockMerits   int64
}

//Kind640202 STATUS:DRAFT
//Used for transferring Shares to another account
type Kind640202 struct {
	Amount    int64
	ToAccount string
	Note      string
}

//Kind640208 STATUS:DRAFT
//Used for creating a new cap table for a rocket
type Kind640208 struct {
	RocketID library.RocketID `json:"rocket_id"`
}

type Mapped map[library.RocketID]map[library.Account]Merit
