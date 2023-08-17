package blocks

import (
	"time"

	"github.com/sasha-s/go-deadlock"
)

var currentState = make(Mapped)
var currentStateMu = &deadlock.Mutex{}

func Tip() (t Block) {
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	return tip()
}

func tip() (t Block) {
	t = Block{
		Height:     800000,
		Hash:       "00000000000000000002a7c4c1e48d76c5a37902165a270156b7a8d72728a054",
		MedianTime: time.Unix(1690165851, 0),
		MinerTime:  time.Unix(1690168629, 0),
		Difficulty: 53911173001054,
	}
	for _, block := range currentState {
		if block.Height > t.Height {
			t = block
		}
	}
	return
}

func getMapped() Mapped {
	return currentState
}
