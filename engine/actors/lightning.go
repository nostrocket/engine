package actors

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/fiatjaf/go-lnurl"
	"github.com/nbd-wtf/go-nostr"
	decodepay "github.com/nbd-wtf/ln-decodepay"
	"nostrocket/engine/library"
)

func DecodeInvoice(invoice string) (b decodepay.Bolt11, e error) {
	bolt11, err := decodepay.Decodepay(invoice)
	if err != nil {
		return b, err
	}
	return bolt11, e
}

func GetLightningAddressFromKind0(event nostr.Event) (string, bool) {
	if len(event.Content) > 0 {
		var profile library.Profile
		err := json.Unmarshal([]byte(event.Content), &profile)
		if err == nil {
			addr, err := mail.ParseAddress(profile.Lud16)
			if err == nil {
				return strings.Trim(addr.String(), "<>"), true
			}
		}
	}
	return "", false
}

func GetInvoice(address string, amount int64, description string) (string, error) {
	return getInvoice(address, amount, description)
}

type LNServicePayResponse struct {
	Callback    string `json:"callback"`
	MaxSendable int64  `json:"maxSendable"`
	MinSendable int64  `json:"minSendable"`
	Metadata    string `json:"metadata"`
	Tag         string `json:"tag"`
}

type LNServiceInvoice struct {
	Pr     string     `json:"pr"`
	Routes []struct{} `json:"routes"`
}

func GetLNServiceResponse(lnurla string) (l LNServicePayResponse, b bool) {
	// Decode LN Url
	decodedLnUrl, _ := lnurl.LNURLDecode(lnurla)
	// Get LN Service URL
	resp, err := http.Get(decodedLnUrl)
	if err != nil {
		LogCLI(err, 2)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LogCLI(err, 2)
		return
	}
	// extract callback URL
	var response LNServicePayResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		LogCLI(err, 2)
		return
	}
	return response, true
}

func decode(url string, amount int64, comment string) (invoice string) {
	// Decode LN Url
	decodedLnUrl, _ := lnurl.LNURLDecode(url)
	// Get LN Service URL
	resp, err := http.Get(decodedLnUrl)
	if err != nil {
		LogCLI(err, 2)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LogCLI(err, 2)
		return
	}
	// extract callback URL
	var response LNServicePayResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		LogCLI(err, 2)
		return
	}
	// Make an HTTP GET request to callback URL with amount to be paid
	// LN Service will create and return a lnd invoice to be paid with this amount
	callbackUrl := response.Callback + "?amount=" + strconv.Itoa(int(amount)) + "&comment=" + strings.TrimSpace(comment)
	resp, err = http.Get(callbackUrl)
	if err != nil {
		LogCLI(err, 2)
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		LogCLI(err, 2)
		return ""
	}

	//fmt.Println(string(body))
	var resInvoice LNServiceInvoice
	err = json.Unmarshal(body, &resInvoice)
	if err != nil {
		LogCLI(err, 2)
		return ""
	}
	return resInvoice.Pr
}

func lud16ToUrl(address string) (s string, e error) {
	split := strings.Split(address, "@")
	if len(split) != 2 {
		e = fmt.Errorf("invalid lightning address")
	}
	return "https://" + strings.Trim(split[1], "<>") + "/.well-known/lnurlp/" + strings.Trim(split[0], "<>"), e
}

func urlToLud06(url string) string {
	encodedUrl, err := lnurl.Encode(url)
	if err != nil {
		LogCLI(err, 1)
	}
	return encodedUrl
}

func getInvoice(address string, amount int64, description string) (string, error) {
	if url, err := lud16ToUrl(address); err == nil {
		lud06 := urlToLud06(url)
		invoice := decode(lud06, amount*1000, description)
		if invoice != "" {
			return invoice, nil
		}
	}
	return "", fmt.Errorf("failed m89u89u")
}

func Lud16ToLud06(lud16 string) (string, bool) {
	url, err := lud16ToUrl(lud16)
	if err != nil {
		LogCLI(err, 1)
		return "", false
	}
	lud06 := urlToLud06(url)
	if len(lud06) > 0 {
		return lud06, true
	}
	return "", false
}
