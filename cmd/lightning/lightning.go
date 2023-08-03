package main

import (
	"fmt"
	"os"
	"strconv"

	"nostrocket/engine/actors"
	"nostrocket/state/payments"
)

func main() {
	if len(os.Args) > 2 {
		address := os.Args[1]
		amount, err := strconv.Atoi(os.Args[2])
		if err != nil {
			actors.LogCLI("invalid amount", 1)
			return
		}
		inv, err := payments.GetInvoice(address, int64(amount), "test with space")
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
