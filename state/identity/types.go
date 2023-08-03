package identity

import (
	"github.com/nbd-wtf/go-nostr"
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
	LatestKind0           nostr.Event
}
