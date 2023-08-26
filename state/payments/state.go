package payments

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/messaging/blocks"
	"nostrocket/messaging/relays"
	"nostrocket/state"
	"nostrocket/state/identity"
	"nostrocket/state/merits"
	"nostrocket/state/replay"
)

var paymentRequests nextPaymentRequest
var products productMap
var paymentRecieved paymentsReceived
var handledReceipts = make(map[library.Sha256]int64)

var currentStateMu = &deadlock.Mutex{}

var started bool = false

func start() {
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	if !started {
		started = true
		paymentRequests = make(map[library.RocketID]map[library.Sha256]PaymentRequest)
		paymentRecieved = make(paymentsReceived)
		products = make(map[library.RocketID]map[library.Sha256]Product)
	}
}

func handleNewPaymentRequest(event nostr.Event) (m Mapped, e error) {
	//todo find existing and don't update unless the account is different, amount different, or has been paid.
	//todo archive the existing one if it has been paid
	//zapdataString, ok := library.GetOpData(event, "zapdata")
	//if ok {
	//	zapData := ZapData{}
	//	err := json.Unmarshal([]byte(zapdataString), &zapData)
	//	if err != nil {
	//		return Mapped{}, err
	//	}
	//	if _, ok := paymentRecieved[zapData.Product.RocketID]; !ok {
	//		paymentRecieved[zapData.Product.RocketID] = make(map[library.Sha256][]PaymentRequest)
	//	}
	//	if _, ok := paymentRecieved[zapData.Product.RocketID][zapData.Product.UID]; !ok {
	//		paymentRecieved[zapData.Product.RocketID][zapData.Product.UID] = []PaymentRequest{}
	//	}
	//	paymentRecieved[zapData.Product.RocketID][zapData.Product.UID] =
	//		append(paymentRecieved[zapData.Product.RocketID][zapData.Product.UID],
	//			paymentRequests[zapData.Product.RocketID][zapData.Product.UID])
	//
	//	paymentRequests[zapData.Product.RocketID][zapData.Product.UID] = PaymentRequest{}
	//}

	//invoice, ok := library.GetOpData(event, "invoice")
	//if !ok {
	//	return m, fmt.Errorf("does not contain an invoice")
	//}
	//decodedInvoice, err := actors.DecodeInvoice(invoice)
	//if err != nil {
	//	return Mapped{}, err
	//}
	product, ok := library.GetOpData(event, "product")
	if !ok {
		return m, fmt.Errorf("does not contain a product")
	}
	productObject, ok := findProductByID(product)
	if !ok {
		return m, fmt.Errorf("product %s could not be found in the current state", product)
	}
	amt, ok := library.GetOpData(event, "amount")
	if !ok {
		return m, fmt.Errorf("does not contain an amount")
	}
	amount, err := strconv.ParseInt(amt, 10, 64)
	if err != nil {
		return m, err
	}
	//if decodedInvoice.MSatoshi/1000 != amount {
	//	return m, fmt.Errorf("amount on invoice is %d but amount on request is %d", decodedInvoice.MSatoshi/1000, amount)
	//}
	if productObject.Amount != amount {
		return m, fmt.Errorf("amount in payment request does not match amount in product")
	}
	account, ok := library.GetOpData(event, "pubkey")
	if !ok {
		return m, fmt.Errorf("does not contain the pubkey of the recieving account")
	}
	nextPaymentAccount, err := merits.GetNextPaymentAddress(productObject.RocketID, productObject.Amount)
	if err != nil {
		return m, err
	}
	if nextPaymentAccount != account {
		return m, fmt.Errorf("account specified by the payment request is not the next payment account")
	}
	lud16, ok := library.GetOpData(event, "lud16")
	if !ok {
		return m, fmt.Errorf("does not contain a lud16")
	}
	lud06, ok := actors.Lud16ToLud06(lud16)
	if !ok {
		return m, fmt.Errorf("could not get lud06")
	}
	lud06FromEvent, ok := library.GetOpData(event, "lud06")
	if !ok {
		//return m, fmt.Errorf("does not contain a lud06")
	}
	if lud06FromEvent != lud06 {
		//return m, fmt.Errorf("lud16 in event does not match query")
	}
	//paymenthash, ok := library.GetOpData(event, "paymenthash")
	//if !ok {
	//	return m, fmt.Errorf("does not contain a payment hash")
	//}
	//if paymenthash != decodedInvoice.PaymentHash {
	//	return m, fmt.Errorf("payment hash in invoice does not match event")
	//}
	if event.PubKey != account {
		//this payment request was created on behalf of the merit holder but not the merit holder themselves
		maintainerOnRocket := state.IsMaintainerOnRocket(event.PubKey, productObject.RocketID)
		//return m, fmt.Errorf("account %s is not a maintainer on this rocket", event.PubKey)
		votePowerInRocket := merits.VotepowerInRocketForAccount(event.PubKey, productObject.RocketID) > 0
		//return m, fmt.Errorf("account %s does not have any votepower in this rocket", event.PubKey)
		maintainer := identity.IsMaintainer(event.PubKey)
		votepower := merits.VotepowerInNostrocketForAccount(event.PubKey) > 0
		if !(maintainerOnRocket || maintainer) && (votepower || votePowerInRocket) {
			fmt.Printf("\nmaintainerOnRocket: %b\nmaintainer: %b\nvotepower: %b\nvotePowerInRocket: %b\n",
				maintainerOnRocket, maintainer, votepower, votePowerInRocket)
			return m, fmt.Errorf("account %s does not have credentials to create payment requests on behalf of others", event.PubKey)
		}
		//validate that the person creating this payment request used the correct lud16
		//todo problem: this this will break if anyone changes their lightning address
		kind0, ok := getLatestKind0(account)
		if !ok {
			return m, fmt.Errorf("could not find a kind 0 event for %s", account)
		}
		lnaddress, ok := actors.GetLightningAddressFromKind0(kind0)
		if !ok {
			return m, fmt.Errorf("could not derive lnaddress from event %s", kind0.ID)
		}
		if lnaddress != lud16 {
			return m, fmt.Errorf("payment request uses %s but the profile of this pubkey lists %s", lud16, lnaddress)
		}
		//todo validate nostr
	}
	lnService, ok := actors.GetLNServiceResponse(lud06)
	if !ok {
		return m, fmt.Errorf("could not get lnservice details")
	}
	if (lnService.MaxSendable / 1000) < amount {
		return m, fmt.Errorf("next payment request is %d but max sendable for this user is %d", amount, lnService.MaxSendable/1000)
	}
	fmt.Println(lnService.LSPubkey)
	paymentRequest := PaymentRequest{
		UID:             event.ID,
		RocketID:        productObject.RocketID,
		ProductID:       productObject.UID,
		WitnessedHeight: blocks.Tip().Height,
		PaidBy:          "",
		AmountPaid:      0,
		AmountRequired:  amount,
		MeritHolder:     account,
		LUD16:           lud16,
		LUD06:           lud06,
		CallbackURL:     lnService.Callback,
		LSPubkey:        lnService.LSPubkey,
		//Invoice:         invoice,
		//PaymentHash:     paymenthash,
	}
	existingRocket, ok := paymentRequests[productObject.RocketID]
	if !ok {
		existingRocket = make(map[library.Sha256]PaymentRequest)
	}
	existingRocket[productObject.UID] = paymentRequest
	paymentRequests[productObject.RocketID] = existingRocket
	return getMapped(), nil
}

func getLatestKind0(account library.Account) (nostr.Event, bool) {
	var kind0 nostr.Event
	actors.LogCLI(fmt.Sprintf("fetching profile for account %s", account), 4)
	if kind0FromRelay, ok := relays.FetchLatestProfile(account); ok {
		kind0 = kind0FromRelay
	}
	if kind0FromState, ok := identity.GetLatestKind0(account); ok {
		if kind0FromState.CreatedAt.Time().After(kind0.CreatedAt.Time()) {
			kind0 = kind0FromState
		}
	}
	if ok, _ := kind0.CheckSignature(); ok {
		return kind0, true
	}
	return nostr.Event{}, false
}

// create an event that generates a new payment request which is to be used to create a zap request
func createPaymentRequestEvent(product Product, zapData ZapData, r_override string) (n nostr.Event, e error) {
	account, err := merits.GetNextPaymentAddress(product.RocketID, product.Amount)
	if err != nil {
		return n, err
	}
	kind0, ok := getLatestKind0(account)
	if !ok {
		return n, fmt.Errorf("could not find latest kind 0 event for account %s", account)
	}
	lud16, ok := actors.GetLightningAddressFromKind0(kind0)
	if !ok {
		fmt.Printf("%#v", kind0)
		return n, fmt.Errorf("could not derive lud16 from event")
	}
	//actors.LogCLI(fmt.Sprintf("fetching a new lightning invoice for address %s", lud16), 4)
	//invoice, err := actors.GetInvoice(lud16, product.Amount, product.UID)
	//if err != nil {
	//	return n, err
	//}
	//decoded, err := actors.DecodeInvoice(invoice)
	//if err != nil {
	//	return n, err
	//}
	lud06, ok := actors.Lud16ToLud06(lud16)
	if !ok {
		return n, fmt.Errorf("could not get lud06")
	}
	n.CreatedAt = nostr.Timestamp(time.Now().Unix())
	n.PubKey = actors.MyWallet().Account
	n.Kind = 15173340
	tags := nostr.Tags{}
	if len(r_override) == 64 {
		tags = append(tags, nostr.Tag{"r", r_override})
	} else {
		tags = append(tags, nostr.Tag{"r", replay.GetCurrentHashForAccount(actors.MyWallet().Account)})
	}

	//tags = append(tags, nostr.Tag{"op", "payments.newrequest.invoice", invoice})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.rocket", product.RocketID})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.product", product.UID})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.amount", fmt.Sprintf("%d", product.Amount)})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.pubkey", account})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.lud16", lud16})
	tags = append(tags, nostr.Tag{"op", "payments.newrequest.lud06", lud06})
	//tags = append(tags, nostr.Tag{"op", "payments.newrequest.paymenthash", decoded.PaymentHash})
	tags = append(tags, nostr.Tag{"e", product.UID, "", "reply"})
	tags = append(tags, nostr.Tag{"e", actors.IgnitionEvent, "", "root"})
	//if we have zapdata, add it to the event
	if len(zapData.PayerPubkey) == 64 {

	}

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

//func createNextPaymentRequests() (pm nextPaymentRequest) {
//	pm = make(nextPaymentRequest)
//	for id, m := range productMap {
//		nextPaymentRequest := make(map[library.Sha256]PaymentRequest)
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
