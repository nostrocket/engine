package payments

import (
	"nostrocket/engine/library"
	"nostrocket/state/merits"

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

func createPaymentRequest(product Product) (p PaymentRequest, e error) {
	address, err := merits.GetNextPaymentAddress(product.RocketID, product.Amount)
	if err != nil {
		return p, err
	}
	p.RocketID = product.RocketID
	p.AmountRequired = product.Amount
	p.MeritHolder = address
	p.ProductID = product.UID
	//fetch invoice
	return
}

//func createNextPaymentRequests() (pm paymentMap) {
//	pm = make(paymentMap)
//	for id, m := range productMap {
//		paymentMap := make(map[library.Sha256]PaymentRequest)
//		for sha256, product := range m {
//			address, err := merits.GetNextPaymentAddress(product.RocketID, product.Amount)
//			if err != nil {
//				actors.LogCLI(err.Error(), 1)
//			}
//
//		}
//	}
//
//}
