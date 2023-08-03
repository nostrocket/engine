package payments

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/fiatjaf/go-lnurl"
	"nostrocket/engine/actors"
)

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

func decode(url string, amount int64, comment string) (invoice string) {
	// Decode LN Url
	decodedLnUrl, _ := lnurl.LNURLDecode(url)
	// Get LN Service URL
	resp, err := http.Get(decodedLnUrl)
	if err != nil {
		actors.LogCLI(err, 2)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		actors.LogCLI(err, 2)
		return
	}
	// extract callback URL
	var response LNServicePayResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		actors.LogCLI(err, 2)
		return
	}
	// Make an HTTP GET request to callback URL with amount to be paid
	// LN Service will create and return a lnd invoice to be paid with this amount
	callbackUrl := response.Callback + "?amount=" + strconv.Itoa(int(amount)) + "&comment=" + strings.TrimSpace(comment)
	resp, err = http.Get(callbackUrl)
	if err != nil {
		actors.LogCLI(err, 2)
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		actors.LogCLI(err, 2)
		return ""
	}

	//fmt.Println(string(body))
	var resInvoice LNServiceInvoice
	err = json.Unmarshal(body, &resInvoice)
	if err != nil {
		actors.LogCLI(err, 2)
		return ""
	}
	return resInvoice.Pr
}

func lightningAddressToUrl(address string) (s string, e error) {
	split := strings.Split(address, "@")
	if len(split) != 2 {
		e = fmt.Errorf("invalid lightning address")
	}
	return "https://" + split[1] + "/.well-known/lnurlp/" + split[0], e
}

func generate(url string) string {
	encodedUrl, err := lnurl.Encode(url)
	if err != nil {
		actors.LogCLI(err, 1)
	}
	return encodedUrl
}

func getInvoice(address string, amount int64, description string) (string, error) {
	if url, err := lightningAddressToUrl(address); err == nil {
		lnurlo := generate(url)
		invoice := decode(lnurlo, amount*1000, description)
		if invoice != "" {
			return invoice, nil
		}
	}
	return "", fmt.Errorf("failed m89u89u")
}
