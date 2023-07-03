package identity

import (
	"nostrocket/engine/library"
)

type Mapped map[library.Account]Identity

type Identity struct {
	Account               library.Account
	Name                  string
	UniqueSovereignBy     library.Account // the account that has validated this account is a real human and doesn't already have an account (identity chain)
	CharacterVouchedForBy map[library.Account]struct{}
	MaintainerBy          library.Account // the maintainer who has made this account a maintainer (maintainer chain)
	Pubkeys               []string
	OpReturnAddr          [][]string
	Order                 int64
	PermanymEventID       library.Sha256
}

type Kind0 struct {
	Name        string `json:"name"`
	About       string `json:"about"`
	DisplayName string `json:"display_name"`
}

//Evolution of public contracts: we can experiment behind this API by keeping Kind640400 (a Nostr Kind that is hereby
//defined by it's JSON unmarshalling struct) and adding new things like DisplayName or KittenNames. Any new things
//MUST be optional once this progresses past DRAFT status, if we need to add mandatory items under this Kind we need a new Kind.

//Kind640402 STATUS:DRAFT
//Used for adding Participants to the USH and Maintainer tree.
type Kind640402 struct {
	Target     library.Account `json:"target"`
	Maintainer bool            `json:"maintainer"`
	USH        bool            `json:"ush"`
	Character  bool            `json:"character"`
}

//Kind640404 STATUS:DRAFT
//Used for adding Identity evidence if the Participant wants to dox themselves
type Kind640404 struct {
	Platform string `json:"platform"`
	Username string `json:"username"`
	Evidence string `json:"evidence"`
}

//Kind640406 STATUS:DRAFT
//Used for adding an OP_RETURN address
type Kind640406 struct {
	Address string
	Proof   string //Bitcoin signed message of the user's pubkey
}
