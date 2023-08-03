package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/fiatjaf/go-lnurl"
	"nostrocket/engine/actors"
)

type LNServicePayResponse struct {
	Callback    string `json:"callback"`
	MaxSendable int    `json:"maxSendable"`
	MinSendable int    `json:"minSendable"`
	Metadata    string `json:"metadata"`
	Tag         string `json:"tag"`
}

type LNServiceInvoice struct {
	Pr     string     `json:"pr"`
	Routes []struct{} `json:"routes"`
}

func decode(url string, amount int, comment string) (invoice string) {
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
	callbackUrl := response.Callback + "?amount=" + strconv.Itoa(amount) + "&comment=" + strings.TrimSpace(comment)
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

func getInvoice(address string, amount int, description string) (string, error) {
	if url, err := lightningAddressToUrl(address); err == nil {
		lnurlo := generate(url)
		invoice := decode(lnurlo, amount*1000, description)
		if invoice != "" {
			return invoice, nil
		}
	}
	return "", fmt.Errorf("failed m89u89u")
}

func main() {
	if len(os.Args) > 2 {
		address := os.Args[1]
		amount, err := strconv.Atoi(os.Args[2])
		if err != nil {
			actors.LogCLI("invalid amount", 1)
			return
		}
		inv, err := getInvoice(address, amount, "test with space")
		if err != nil {
			actors.LogCLI(err.Error(), 1)
			return
		}
		actors.LogCLI(inv, 4)
		return
	} else {
		fmt.Printf("\nUsage example:\n./lightning gsovereignty@getalby.com 5000\n")
	}
}
