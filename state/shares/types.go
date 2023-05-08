package shares

import (
	"nostrocket/engine/library"
)

type Share struct {
	LeadTimeLockedShares   int64
	LeadTime               int64 // in multiples of 2016 blocks (2 weeks)
	LastLtChange           int64 // btc height
	LeadTimeUnlockedShares int64 //Approved Expenses can be swept into here and sold even if Participant's LT is >0
	OpReturnAddresses      []string
}

//Kind640200 STATUS:DRAFT
//Used for adjusting lead time
type Kind640200 struct {
	AdjustLeadTime string //+ or -
	LockShares     int64
	UnlockShares   int64
}

//Kind640202 STATUS:DRAFT
//Used for transferring Shares to another account
type Kind640202 struct {
	Amount    int64
	ToAccount string
	Note      string
}

//Kind640208 STATUS:DRAFT
//Used for creating a new cap table for a mirv
type Kind640208 struct {
	RocketID library.RocketID `json:"rocket_id"`
}

type Mapped map[library.RocketID]map[library.Account]Share
