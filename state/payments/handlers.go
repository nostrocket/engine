package payments

import (
	"fmt"
	"strconv"
	"strings"

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
	default:
		return m, fmt.Errorf("I am the payments mind, event %s was sent to me but I don't know how to handle kind %d", event.ID, event.Kind)
	}
}

func handleByTags(event nostr.Event) (m Mapped, e error) {
	if operation, ok := library.GetFirstTag(event, "op"); ok {
		ops := strings.Split(operation, ".")
		if len(ops) > 2 {
			if ops[1] == "payments" {
				if err := validateRocket(event); err != nil {
					return Mapped{}, err
				}
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
	return Mapped{}, fmt.Errorf("event %s is attempting to change the state of products but did not contain a valid opcode", event.ID)
}

// create new product
func createProduct(event nostr.Event) (m Mapped, err error) {
	amount, ok := library.GetOpData(event, "amount")
	if !ok {
		return m, fmt.Errorf("event %s wants to create a new product but does not contain an amount", event.ID)
	}
	sats, e := strconv.ParseInt(amount, 10, 64)
	if e != nil {
		return m, fmt.Errorf("event %s wants to create a new product but converting string to int failed with error %s", event.ID, e.Error())
	}
	infoID, ok := library.GetOpData(event, "info")
	if !ok {
		return m, fmt.Errorf("event %s wants to create a new product but does not contain an event ID for product information", event.ID)
	}
	rocketID, ok := library.GetOpData(event, "rocket")
	if !ok {
		return m, fmt.Errorf("event %s wants to create a new product but does not contain a reocket ID", event.ID)
	}
	existingRocketProducts, exists := products[rocketID]
	if !exists {
		existingRocketProducts = make(map[library.Sha256]Product)
	}
	existingRocketProducts[event.ID] = Product{
		UID:                event.ID,
		RocketID:           rocketID,
		Amount:             sats,
		ProductInformation: infoID,
	}
	products[rocketID] = existingRocketProducts
	return getMapped(), nil
}

//modify existing product

//notify payment made
