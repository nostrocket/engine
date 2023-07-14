package merits

import (
	"nostrocket/engine/library"
)

//Merit STATUS:DRAFT
type Merit struct {
	RocketID               library.RocketID
	LeadTimeLockedMerits   int64
	LeadTime               int64 // in multiples of 2016 blocks (2 weeks)
	LastLtChange           int64 // btc height
	LeadTimeUnlockedMerits int64 //Approved Expenses can be swept into here and sold even if Participant's LT is >0
	OpReturnAddresses      []string
	Requests               []Request
}

//Request STATUS:DRAFT
type Request struct {
	RocketID          library.RocketID
	UID               library.Sha256
	Problem           library.Sha256 //ID of problem in the problem tracker (optional)
	CommitMsg         string         //The commit message from a merged patch on github or the patch chain. MUST NOT exceed 100 chars
	Solution          library.Sha256 //For now, this MUST be a sha256 hash of the diff that is merged on github, later this hash will be used in the patch chain.
	Commit            string         //A link to a commit. This SHOULD be deprecated as soon as possible and replaced with a Nostrocket object instead of an external link
	Amount            int64          //Satoshi
	RepaidAmount      int64
	WitnessedAt       int64 //Height at which we saw this Request being created
	Nth               int64 //Requests are repaid in the order they were approved, this is the 'height' of this Request, it is added upon approval
	Ratifiers         map[library.Account]struct{}
	RatifyPermille    int64
	Blackballers      map[library.Account]struct{}
	BlackballPermille int64
	Approved          bool
	Rejected          bool
	MeritsCreated     int64 //should be equal to the Amount in satoshi
}

type Mapped map[library.RocketID]map[library.Account]Merit
