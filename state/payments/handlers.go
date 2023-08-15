package payments

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/state"
	"nostrocket/state/identity"
	"nostrocket/state/merits"
	"nostrocket/state/replay"

	"github.com/nbd-wtf/go-nostr"
)

func HandleEvent(event nostr.Event) (m Mapped, err error) {
	//todo when user creates a zap request event, store this and the response from the service provider as evidence in case something goes wrong. Allow them to manually input ths data if they paid but it didn't work.
	start()
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	switch event.Kind {
	case 1:
		return handleByTags(event)
	case 9735:
		//fmt.Printf("\n%#v\n", event)
		return handleZapReceipt(event)
		//return Mapped{}, fmt.Errorf("not implemented")
		//todo handle zaps
		//when we see a zap reciept, produce an event containing the zap reciept, same way we deal with 3340 events.
		//client side is always presented with the next event to zap to pay for a product (each merit holder publishes an event for incoming payments)
		//whenever we see a zap we add this to the merit holder's payment recieved records and factor it into the round robin
		//if payments come in too fast its no big deal, no need to verify that the payments are going to the right place, just tally them and update state and let client side figure it out.
	case 3340:
		return handleNewPaymentRequest(event)
	case 15179735:
		return handlePaymentProof(event)
	default:
		return m, fmt.Errorf("I am the payments mind, event %s was sent to me but I don't know how to handle kind %d", event.ID, event.Kind)
	}
}

func handlePaymentProof(event nostr.Event) (m Mapped, e error) {
	zapData, err := parseAndValidatePaymentReceipt(event)
	if err != nil {
		return Mapped{}, err
	}
	if _, exists := handledReceipts[zapData.ZapReceiptID]; exists {
		return Mapped{}, fmt.Errorf("already handled this zap")
	}
	if vp := merits.VotepowerInNostrocketForAccount(event.PubKey); vp < 1 && zapData.PayeePubkey != event.PubKey {
		return Mapped{}, fmt.Errorf("payment proof is not signed by votepower or payee")
	}

	//add the paid amount to the merits mind and add the updated state to the return object
	meritsMapped, err := merits.HandleIncomingPayment(zapData.PayeePubkey, zapData.Amount, zapData.Product.RocketID, zapData.ZapReceiptID)
	if err != nil {
		return Mapped{}, err
	}
	m.Outbox = append(m.Outbox, meritsMapped)
	//add the payer to the product
	//todo make this the bitcoin height or bitcoin height + payment period (for expiry)
	if len(products[zapData.Product.RocketID][zapData.Product.UID].CurrentUsers) == 0 {
		existing := products[zapData.Product.RocketID][zapData.Product.UID]
		existing.CurrentUsers = make(map[library.Account]int64)
		products[zapData.Product.RocketID][zapData.Product.UID] = existing
	}
	products[zapData.Product.RocketID][zapData.Product.UID].CurrentUsers[zapData.PayerPubkey] = 0

	//create an event to archive the payment request if this is the next one
	if paymentRequests[zapData.Product.RocketID][zapData.Product.UID].MeritHolder == zapData.PayerPubkey {
		requestEvent, err := createPaymentRequestEvent(products[zapData.Product.RocketID][zapData.Product.UID], zapData)
		if err != nil {
			actors.LogCLI(err, 1)
		} else {
			m.Outbox = append(m.Outbox, requestEvent)
		}
	}

	//update next payment request and archive existing if this one is correct

	//add the zap receipt ID to the handled dataset so we dont do it again
	handledReceipts[zapData.ZapReceiptID] = 0 //todo make this the current bitcoin height
	//fmt.Printf("\n71%#v\n", meritsMapped)
	//fmt.Printf("\n72%#v\n", zapData)
	//return Mapped{}, fmt.Errorf("ssdfsa")
	mapped := getMapped()
	m.Payments = mapped.Payments
	m.Products = mapped.Products
	fmt.Printf("\n%#v\n", m.Products)
	for _, outbox := range m.Outbox {
		fmt.Printf("\n%#v\n", outbox)
	}
	return
}

func handleZapReceipt(event nostr.Event) (m Mapped, e error) {
	if _, exists := handledReceipts[event.ID]; exists {
		return Mapped{}, fmt.Errorf("already handled this zap receipt")
	}
	//todo if we have votepower, we do this. If we don't, then we only handle payment recieved events produced by the recieving pubkey or votepower
	zap, err := validateAndReturnZap(event)
	if err != nil {
		return Mapped{}, err
	}
	//fmt.Printf("\n%#v\n", zap)
	//Create a 97351 event to:
	//add a paymentsReceived record to the product,
	//add the user to the user list,
	//update merits (Call merits to update merits, and forward new state to the event conductor),
	//trigger updating the next payment request if recieve == PaymentRequest. This contains the 9735 event,
	// add zap receipt ID to list so we don't handle it twice
	m = getMapped()
	if vp := merits.VotepowerInNostrocketForAccount(actors.MyWallet().Account); vp > 0 || zap.PayeePubkey == actors.MyWallet().Account {
		zapBytes, err := json.Marshal(event)
		if err != nil {
			return Mapped{}, err
		}
		t := nostr.Tags{}
		t = append(t, nostr.Tag{"r", replay.GetCurrentHashForAccount(actors.MyWallet().Account)})
		t = append(t, nostr.Tag{"e", zap.Product.UID})
		t = append(t, nostr.Tag{"e", actors.IgnitionEvent, "", "root"})
		t = append(t, nostr.Tag{"9735", fmt.Sprintf("%s", zapBytes)})
		t = append(t, nostr.Tag{"p", zap.PayerPubkey})
		t = append(t, nostr.Tag{"p", zap.PayeePubkey})
		r := nostr.Event{
			PubKey:    actors.MyWallet().Account,
			CreatedAt: nostr.Timestamp(time.Now().Unix()),
			Kind:      15179735,
			Tags:      t,
			Content:   fmt.Sprintf("Payment for product nostrocket:%s", zap.Product.UID),
		}
		r.ID = r.GetID()
		err = r.Sign(actors.MyWallet().PrivateKey)
		if err != nil {
			return Mapped{}, err
		}
		m.Outbox = append(m.Outbox, r)
	}
	return m, nil
}

func handleByTags(event nostr.Event) (m Mapped, e error) {
	if operation, ok := library.GetFirstTag(event, "op"); ok {
		ops := strings.Split(operation, ".")
		if len(ops) > 2 {
			if ops[1] == "payments" {
				switch o := ops[2]; {
				case o == "product":
					return handleProduct(event)
				case o == "newrequest":
				case o == "vote":
				}
			}
		}
	}
	return m, fmt.Errorf("no valid operation found 35645ft")
}

func validateRocket(event nostr.Event) error {
	rocketID, ok := library.GetOpData(event, "rocket")
	if !ok {
		return fmt.Errorf("event %s does not contain a rocket ID", event.ID)
	}
	if _, ok := state.Rockets()[rocketID]; !ok {
		return fmt.Errorf("event %s contains invalid rocket ID %s", event.ID, rocketID)
	}
	return nil
}

func handleProduct(event nostr.Event) (m Mapped, err error) {
	if !identity.IsUSH(event.PubKey) {
		return m, fmt.Errorf("event %s: pubkey %s not in identity tree", event.ID, event.PubKey)
	}
	_, ok := library.GetOpData(event, "new")
	if ok {
		return createProduct(event)
	}
	_, ok = library.GetOpData(event, "modify")
	if ok {
		return modifyProduct(event)
	}
	return Mapped{}, fmt.Errorf("event %s is attempting to change the state of products but did not contain a valid opcode", event.ID)
}

// create new product
func createProduct(event nostr.Event) (m Mapped, err error) {
	amount, err := validateAndReturnOpcodeData(event, "amount")
	if err != nil {
		return m, fmt.Errorf("%s tried to create a new product but %s", event.ID, err.Error())
	}
	sats := amount[0].(int64)
	infoID, ok := library.GetOpData(event, "info")
	if !ok {
		return m, fmt.Errorf("event %s wants to create a new product but does not contain an event ID for product information", event.ID)
	}
	rocketQuery, err := validateAndReturnOpcodeData(event, "rocket")
	if err != nil {
		return m, fmt.Errorf("%s attempted to create a new product but %s", event.ID, err.Error())
	}
	rocketID := rocketQuery[0].(string)
	if !state.IsMaintainerOnRocket(event.PubKey, rocketID) {
		return m, fmt.Errorf("%s wants to create a product but not signed by a pubkey who is a maintainer on this rocket", event.ID)
	}
	existingRocketProducts := rocketQuery[1].(map[library.Sha256]Product)
	existingRocketProducts[event.ID] = Product{
		UID:                event.ID,
		RocketID:           rocketID,
		Amount:             sats,
		ProductInformation: infoID,
	}
	products[rocketID] = existingRocketProducts
	mapped := getMapped()
	mapped.Outbox = append(mapped.Outbox, nostr.Event{Kind: 15171031, Content: event.ID})
	//instead of this, look for a kind0 and package it in a new event along with the invoice.
	//we should actually get the next-payment-address account to do this instead but for now just trust anyone with votepower
	//so the state of payment is not updated in consensus unless the kind0 is packaged, this solves the problem of missing kind0s during consensus formation
	if !event.GetExtra("fromConsensusEvent").(bool) {
		request, err := createPaymentRequestEvent(existingRocketProducts[event.ID], ZapData{})
		if err != nil {
			actors.LogCLI(err.Error(), 1)
		} else {
			mapped.Outbox = append(mapped.Outbox, request)
			//fmt.Printf("%#v", request)
		}
	}
	return mapped, nil
}

// modify existing product
func modifyProduct(event nostr.Event) (m Mapped, err error) {
	targetData, err := validateAndReturnOpcodeData(event, "target")
	if err != nil {
		return m, fmt.Errorf("%s wants to modify a product, but %s", event.ID, err.Error())
	}
	target := targetData[0].(library.Sha256)
	existingProduct := targetData[1].(Product)
	existingRocketProducts := targetData[2].(map[library.Sha256]Product)
	if !state.IsMaintainerOnRocket(event.PubKey, existingProduct.RocketID) {
		return m, fmt.Errorf("%s wants to modify a product but not signed by a pubkey who is a maintainer on this rocket", event.ID)
	}
	var updates = 0
	if amount, err := validateAndReturnOpcodeData(event, "amount"); err == nil {
		sats := amount[0].(int64)
		if sats > 0 {
			if existingProduct.Amount != sats {
				existingProduct.Amount = sats
				updates++
			}
		}
	}
	if infoID, ok := library.GetOpData(event, "info"); ok {
		if len(infoID) == 64 {
			if existingProduct.ProductInformation != infoID {
				existingProduct.ProductInformation = infoID
				updates++
			}
		}
	}
	if updates > 0 {
		existingRocketProducts[target] = existingProduct
		products[existingProduct.RocketID] = existingRocketProducts
		mapped := getMapped()
		if !event.GetExtra("fromConsensusEvent").(bool) {
			request, err := createPaymentRequestEvent(existingProduct, ZapData{})
			if err != nil {
				actors.LogCLI(err.Error(), 1)
			} else {
				mapped.Outbox = append(mapped.Outbox, request)
				//fmt.Printf("%#v", request)
			}
		}

		return mapped, nil
	}
	return m, fmt.Errorf("%s tried to modify a product but did not contain a valid state change", event.ID)
}

func findProductByID(id library.Sha256) (p Product, o bool) {
	for _, m := range products {
		if prod, ok := m[id]; ok {
			return prod, true
		}
	}
	return
}

func validateAndReturnOpcodeData(event nostr.Event, opcode string) (r []any, e error) {
	switch opcode {
	case "target":
		target, ok := library.GetOpData(event, "target")
		if !ok {
			return nil, fmt.Errorf("does not contain a target")
		}
		r = append(r, target)
		product, ok := findProductByID(target)
		if !ok {
			return nil, fmt.Errorf("target product does not exist")
		}
		r = append(r, product)
		rocket, ok := products[product.RocketID]
		if !ok {
			return nil, fmt.Errorf("rocket does not exist")
		}
		r = append(r, rocket)
		return
	case "rocket":
		rocketID, ok := library.GetOpData(event, "rocket")
		if !ok {
			return nil, fmt.Errorf("does not contain a rocket ID")
		}
		r = append(r, rocketID)
		existing, exists := products[rocketID]
		if !exists {
			existing = make(map[library.Sha256]Product)
		}
		r = append(r, existing)
		return
	case "amount":
		amount, ok := library.GetOpData(event, "amount")
		if !ok {
			return r, fmt.Errorf("does not contain an amount")
		}
		sats, err := strconv.ParseInt(amount, 10, 64)
		if err != nil {
			return r, fmt.Errorf("converting amount in string to int failed with error %s", e.Error())
		}
		r = append(r, sats)
		return
	}
	return
}

//todo notify payment made

//EXAMPLE ZAP
//nostr.Event{ID:"711229a871fe1eea5c167aa06eaecb06403ede99f5555b14bfbf39d75c1adcaf", PubKey:"79f00d3f5a19ec806189fcab03c1be4ff81d18ee4f653c88fac41fe03570f432", CreatedAt:1691739525, Kind:9735, Tags:nostr.Tags{nostr.Tag{"p", "d91191e30e00444b942c0e82cad470b32af171764c2275bee0bd99377efd4075"}, nostr.Tag{"e", "734a503bd5379275e26b88538024845f0904401c1bee8d0d80df800d911f19b9"}, nostr.Tag{"bolt11", "lnbc100n1pjdtet7pp5jcmv7hv407ksxgjjzh4s0df8umt7y638vyp89jqajd6funwu27vqhp5ks2n3pyktnm59kza8aflanj2gfv046qxlmc0flkqs6ucp7lznfmqcqzzsxqyz5vqsp5qv5sdkv43p724x0a4e6rndm8n8vy43yyprw044hn8uu4hx7d3req9qyyssq2v7pw0ld0tzaua9lg2jweuvht6td8sz0p5uj5nzvj5vxmdfz9sspq6n67usen5ngsj78k3hwfzf0zt4yp5ha7pwtwaej00ak2ahlzegpxs4avn"}, nostr.Tag{"preimage", "3e2bbb0dfe52a0194daf52ddaeef389052ca2a7b9559766ef3f404727f760f3b"}, nostr.Tag{"description", "{\"created_at\":1691739517,\"content\":\"I'm paying for a Nostrocket product\",\"tags\":[[\"relays\",\"wss://nostr.mutinywallet.com\"],[\"amount\",\"10000\"],[\"lnurl\",\"LNURL1DP68GURN8GHJ7EM9W3SKCCNE9E3K7MF09EMK2MRV944KUMMHDCHKCMN4WFK8QTM8WDHHVETJV45KWMN50Y7WTHX2\"],[\"p\",\"d91191e30e00444b942c0e82cad470b32af171764c2275bee0bd99377efd4075\"],[\"e\",\"734a503bd5379275e26b88538024845f0904401c1bee8d0d80df800d911f19b9\"]],\"kind\":9734,\"pubkey\":\"d91191e30e00444b942c0e82cad470b32af171764c2275bee0bd99377efd4075\",\"id\":\"0a4e7787fbdacfdc708699432af53d15f1989bd700e2e3bb70b33142080e9dc8\",\"sig\":\"7034751a766e2b37a8a3a1544e9108788900fee0b5ebe5e82d7109c81c27e88bf52a0fe4e21cf2c2f4155e9a552fbe10f94cdb8e1e67a54196187b09c2af5810\"}"}}, Content:"I'm paying for a Nostrocket product", Sig:"7332c7cecb6ae259ec8ae9d4fe9aa3a3db430bf03b1cbcf864e2bbee1a147d572f3c38c4ded86c299e9db382c970607085ad364907b38f0819a8b44a5555b32f", extra:map[string]interface {}{"fromConsensusEvent":false}}
