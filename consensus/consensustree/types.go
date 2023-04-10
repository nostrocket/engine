package consensustree

import (
	"nostrocket/engine/library"
)

type TreeEvent struct {
	StateChangeEventHeight int64
	StateChangeEventID     library.Sha256
	Signers                []library.Account
	EventIDs               []library.Sha256
	IHaveSigned            bool
	IHaveReplaced          bool
	Votepower              int64
	TotalVotepoweAtHeight  int64
	Permille               int64
	BitcoinHeight          int64
}

type Kind640064 struct {
	StateChangeEventID library.Sha256 `json:"event_id"`
	Height             int64          `json:"height"`
	BitcoinHeight      int64          `json:"bitcoin_height"`
}
