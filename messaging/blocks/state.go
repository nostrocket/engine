package blocks

import (
	"github.com/sasha-s/go-deadlock"
)

var currentState = make(Mapped)
var currentStateMu = &deadlock.Mutex{}

func Tip() (t Block, b bool) {
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	return tip()
}

func tip() (t Block, b bool) {
	for _, block := range currentState {
		if block.Height > t.Height {
			t = block
			b = true
		}
	}
	return
}

func getMapped() Mapped {
	return currentState
}
