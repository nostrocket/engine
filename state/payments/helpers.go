package payments

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

// parseAndValidatePaymentReceipt validates that the zapReceipt is signed by the pubkey of the lightning service provider
// specified by the payee's kind 0 event, validates the product, and amounts, etc.
func parseAndValidatePaymentReceipt(event nostr.Event) (z ZapData, e error) {
	zapReceipt, err := getInnerEvent(event, "9735")
	if err != nil {
		return ZapData{}, err
	}
	return validateAndReturnZap(zapReceipt)
}

func getInnerEvent(event nostr.Event, tag string) (n nostr.Event, e error) {
	eventString, ok := library.GetFirstTag(event, tag)
	if !ok {
		return n, fmt.Errorf("could not find zap request event in description tag")
	}
	eventParsed := nostr.Event{}
	err := json.Unmarshal([]byte(eventString), &eventParsed)
	if err != nil {
		return n, err
	}
	_, err = eventParsed.CheckSignature()
	if err != nil {
		return n, err
	}
	return eventParsed, nil
}

func parseZapReceipt(event nostr.Event) (z ZapData, e error) {
	zapRequest, err := getInnerEvent(event, "description")
	if err != nil {
		return ZapData{}, err
	}
	z.LSPubkey = event.PubKey
	amtString, ok := library.GetFirstTag(zapRequest, "amount")
	if !ok {
		return z, fmt.Errorf("could not get amount")
	}
	amount, err := strconv.ParseInt(amtString, 10, 64)
	if err != nil {
		return ZapData{}, err
	}
	z.Amount = amount / 1000
	z.PayerPubkey = zapRequest.PubKey
	product, err := findProductFromZapReceipt(event)
	if err != nil {
		return ZapData{}, err
	}
	z.Product = product
	if z.Amount < product.Amount {
		return z, fmt.Errorf("amount paid is less than the price of the product")
	}
	payee, ok := library.GetFirstTag(event, "p")
	if !ok {
		return ZapData{}, fmt.Errorf("could not get payee pubkey")
	}
	if len(payee) != 64 {
		return ZapData{}, fmt.Errorf("invalid payee pubkey")
	}
	z.PayeePubkey = payee
	z.ZapReceiptID = event.ID
	return
}

// validateAndReturnZap validate the pubkey matches the user's kind0 lnurl endpoint nostrPubkey, Find the associated product and validate that the amount is correct
func validateAndReturnZap(event nostr.Event) (z ZapData, e error) {
	zp, err := parseZapReceipt(event)
	if err != nil {
		return z, err
	}
	kind0, ok := getLatestKind0(zp.PayerPubkey)
	if !ok {
		return z, fmt.Errorf("could not find a kind 0 event for %s", zp.PayerPubkey)
	}
	lnaddress, ok := actors.GetLightningAddressFromKind0(kind0)
	if !ok {
		return z, fmt.Errorf("could not derive lnaddress from event %s", kind0.ID)
	}
	lud06, ok := actors.Lud16ToLud06(lnaddress)
	if !ok {
		return z, fmt.Errorf("could not get lud06 for %s", lnaddress)
	}
	response, ok := actors.GetLNServiceResponse(lud06)
	if !ok {
		return z, fmt.Errorf("could not get lightning service provider details for %s", lud06)
	}
	if response.LSPubkey != event.PubKey {
		return z, fmt.Errorf("the zap receipt is not signed by the user's lightning service provider")
	}
	return zp, nil
}

func findProductFromZapReceipt(event nostr.Event) (p Product, e error) {
	productID, ok := library.GetFirstTag(event, "e")
	if !ok {
		return Product{}, fmt.Errorf("could not find an e tag")
	}
	for _, m := range products {
		for _, product := range m {
			if product.UID == productID {
				return product, nil
			}
		}
	}
	return Product{}, fmt.Errorf("could not find a product with tag %s", productID)
}
