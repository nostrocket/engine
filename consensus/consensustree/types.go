package consensustree

import (
	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
)

type TreeEvent struct {
	StateChangeEventHeight  int64
	StateChangeEventID      library.Sha256
	StateChangeEventHandled bool
	Signers                 map[library.Account]int64 //votepower
	ConsensusEvents         map[library.Sha256]nostr.Event
	IHaveSigned             bool
	IHaveReplaced           bool
	Votepower               int64
	TotalVotepoweAtHeight   int64
	Permille                int64
	BitcoinHeight           int64
}

type Kind640064 struct {
	StateChangeEventID library.Sha256 `json:"event_id"`
	Height             int64          `json:"height"`
	BitcoinHeight      int64          `json:"bitcoin_height"`
}
