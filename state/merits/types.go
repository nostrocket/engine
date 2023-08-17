package merits

import (
	"nostrocket/engine/library"
)

// Merit STATUS:DRAFT
type Merit struct {
	Account                library.Account
	RocketID               library.RocketID
	LeadTimeLockedMerits   int64 //calculated from LeadTimeLockedRequests
	LeadTime               int64 // in multiples of 2016 blocks (2 weeks)
	LastLtChange           int64 // btc height
	LeadTimeUnlockedMerits int64 //calculated from LeadTimeLockedRequests
	TotalMerits            int64
	Requests               []Request
}

// Request STATUS:DRAFT
type Request struct {
	LtLocked          bool            //indicates if the Merits in this request are Locked under the current lead time or not
	OwnedBy           library.Account //the current owner of these merits
	CreatedBy         library.Account //account that did the work to create these merits
	RocketID          library.RocketID
	UID               library.Sha256
	Problem           library.Sha256 //ID of problem in the problem tracker (optional)
	CommitMsg         string         //The commit message from a merged patch on github or the patch chain. MUST NOT exceed 100 chars
	Solution          library.Sha256 //For now, this MUST be a sha256 hash of the diff that is merged on github, later this hash will be used in the patch chain.
	Commit            string         //A link to a commit. This SHOULD be deprecated as soon as possible and replaced with a Nostrocket object instead of an external link
	Amount            int64          //Satoshi
	RemuneratedAmount int64          //the amount that has been repaid
	DividendAmount    int64          //the amount that has been received as dividends
	WitnessedAt       int64          //Height at which we saw this Request being created
	Nth               int64          //Requests are repaid in the order they were approved, this is the 'height' of this Request, it is added upon approval
	Ratifiers         map[library.Account]struct{}
	RatifyPermille    int64
	Blackballers      map[library.Account]struct{}
	BlackballPermille int64
	Approved          bool
	Rejected          bool
	MeritsCreated     int64
}

type Mapped map[library.RocketID]map[library.Account]Merit

// when you sell a request, nth gets reset to the current height, putting everything on equal terms
// make leadtimelockedmerits a map of request IDs and make a function to calculate the total
// to sell, you send an event with a lightning invoice for the amount requested, and then you append your pubkey to the preimage and rehash it, tagging the seller. The seller's client should then automatically broadcast an event to transfer the request to the buyer.
// if dispute resolution is required this can be conducted by votepower.
