package payments

import (
	"fmt"
	"strconv"
	"strings"

	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/state"
	"nostrocket/state/identity"

	"github.com/nbd-wtf/go-nostr"
)

func HandleEvent(event nostr.Event) (m Mapped, err error) {
	start()
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	switch event.Kind {
	case 1:
		return handleByTags(event)
	case 3340:
		return handleNewPaymentRequest(event)
	case 3341:
		//todo handle payment proof, publish events to find next payment address and replace all payment requests for all products in this rocket
		return
	default:
		return m, fmt.Errorf("I am the payments mind, event %s was sent to me but I don't know how to handle kind %d", event.ID, event.Kind)
	}
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
	//instead of this, look for a kind0 and package it in a new event along with the invoice.
	//we should actually get the next-payment-address account to do this instead but for now just trust anyone with votepower
	//so the state of payment is not updated in consensus unless the kind0 is packaged, this solves the problem of missing kind0s during consensus formation
	request, err := createPaymentRequestEvent(existingRocketProducts[event.ID])
	if err != nil {
		actors.LogCLI(err.Error(), 1)
	} else {
		mapped.Outbox = append(mapped.Outbox, request)
		fmt.Printf("%#v", request)
	}
	//fmt.Printf("%#v", request)
	return mapped, nil
}

//modify existing product
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
		request, err := createPaymentRequestEvent(existingProduct)
		if err != nil {
			actors.LogCLI(err.Error(), 1)
		} else {
			mapped.Outbox = append(mapped.Outbox, request)
			fmt.Printf("%#v", request)
		}
		//fmt.Printf("%#v", request)
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
