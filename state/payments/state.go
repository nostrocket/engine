package payments

import (
	"nostrocket/engine/library"

	"github.com/sasha-s/go-deadlock"
)

var paymentRequests paymentMap
var products productMap

var currentStateMu = &deadlock.Mutex{}

var started bool = false

func start() {
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	if !started {
		started = true
		paymentRequests = make(map[library.RocketID]map[library.Sha256]PaymentRequest)
		products = make(map[library.RocketID]map[library.Sha256]Product)
	}
}
