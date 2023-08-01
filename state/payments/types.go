package payments

import (
	"nostrocket/engine/library"
)

type Product struct {
	//todo use voting to change price and information
	UID                library.Sha256
	RocketID           library.Sha256
	Amount             int64          //price in sats
	ProductInformation library.Sha256 //ID of event with information about the product
}

type PaymentRequest struct {
	UID             library.Sha256
	RocketID        library.Sha256
	ProductID       library.Sha256
	WitnessedHeight int64 //bitcoin height the payment was witnessed at
	PaidBy          library.Account
	AmountPaid      int64
	AmountRequired  int64
}

type productMap map[library.RocketID]map[library.Sha256]Product
type paymentMap map[library.RocketID]map[library.Sha256]PaymentRequest

type Mapped struct {
	Products productMap
	Payments paymentMap
}

func GetMapped() (m Mapped) {
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	return getMapped()
}

func getMapped() (m Mapped) {
	m.Payments = paymentRequests
	m.Products = products
	return
}
