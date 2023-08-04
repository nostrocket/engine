package payments

import (
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/messaging/eventcatcher"
	"nostrocket/state/identity"
	"nostrocket/state/merits"
	"nostrocket/state/replay"
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

//create an event that generates a new payment request
func createPaymentRequestEvent(product Product) (n nostr.Event, e error) {
	address, err := merits.GetNextPaymentAddress(product.RocketID, product.Amount)
	if err != nil {
		return n, err
	}
	var kind0 nostr.Event
	actors.LogCLI(fmt.Sprintf("fetching profile for account %s", address), 4)
	if kind0FromRelay, ok := eventcatcher.FetchLatestKind0([]library.Account{address}); ok {
		kind0 = kind0FromRelay
	}
	if kind0FromState, ok := identity.GetLatestKind0(address); ok {
		if kind0FromState.CreatedAt.After(kind0.CreatedAt) {
			kind0 = kind0FromState
		}
	}
	lnaddress, ok := actors.GetLightningAddressFromKind0(kind0)
	if !ok {
		fmt.Printf("%#v", kind0)
		return n, fmt.Errorf("could not derive lnaddress from event")
	}
	actors.LogCLI(fmt.Sprintf("fetching a new lightning invoice for address %s", lnaddress), 4)
	invoice, err := actors.GetInvoice(lnaddress, product.Amount, product.UID)
	if err != nil {
		return n, err
	}
	decoded, err := actors.DecodeInvoice(invoice)
	if err != nil {
		return n, err
	}
	n.CreatedAt = time.Now()
	n.PubKey = actors.MyWallet().Account
	n.Kind = 3340
	tags := nostr.Tags{}
	tags = append(tags, nostr.Tag{"r", replay.GetCurrentHashForAccount(actors.MyWallet().Account)})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.invoice", invoice})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.rocket", product.RocketID})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.product", product.UID})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.amount", fmt.Sprintf("%d", product.Amount)})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.pubkey", address})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.lud16", lnaddress})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.paymenthash", decoded.PaymentHash})
	tags = append(tags, nostr.Tag{"e", product.UID, "", "reply"})
	tags = append(tags, nostr.Tag{"e", actors.IgnitionEvent, "", "root"})
	n.Tags = tags
	n.Content = ""
	n.ID = n.GetID()
	n.Sign(actors.MyWallet().PrivateKey)
	return n, nil
}

//
////handle events that create new payment requests
//func handlePaymentRequest() {
//	address, err := merits.GetNextPaymentAddress(product.RocketID, product.Amount)
//	if err != nil {
//		return p, err
//	}
//	address, ok := identity.GetLatestKind0(p.MeritHolder)
//	//todo problem: if someone changes their address in future then past payment requests will fail to validate
//
//	if !ok {
//		return p, fmt.Errorf("could not get lightning address from profile")
//	}
//}
//
//func createPaymentRequest(product Product) (p PaymentRequest, e error) {
//	address, err := merits.GetNextPaymentAddress(product.RocketID, product.Amount)
//	if err != nil {
//		return p, err
//	}
//	p.RocketID = product.RocketID
//	p.AmountRequired = product.Amount
//	p.MeritHolder = address
//	p.ProductID = product.UID
//
//	address, ok := identity.GetLatestKind0(p.MeritHolder)
//	if !ok {
//		return p, fmt.Errorf("could not get lightning address from profile")
//	}
//	p.LUD16 = address
//
//	invoice, err := getInvoice(address, p.AmountRequired, p.ProductID)
//	if err != nil {
//		return PaymentRequest{}, err
//	}
//	p.Invoice = invoice
//	//fetch invoice
//	return
//}

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
