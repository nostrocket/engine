package main

import (
	"fmt"

	"github.com/eiannone/keyboard"
	"nostrocket/consensus/identity"
	"nostrocket/consensus/replay"
	"nostrocket/engine/actors"
)

// cliListener is a cheap and nasty way to speed up development cycles. It listens for keypresses and executes commands.
func cliListener(interrupt chan struct{}) {
	fmt.Println("Press:\nq: to quit\ns: to print shares\ni: to print identity\nw: to print your current wallet\nSee cliListener.go for more")
	for {
		r, k, err := keyboard.GetSingleKey()
		if err != nil {
			panic(err)
		}
		str := string(r)
		switch str {
		default:
			if k == 13 {
				fmt.Println("\n-----------------------------------")
				break
			}
			if r == 0 {
				break
			}
			fmt.Println("Key " + str + " is not bound to any test procedures. See main.cliListener for more details.")
		case "q":
			close(interrupt)
		case "w":
			fmt.Printf("Current Wallet: \n%v\n", actors.MyWallet())
		case "I":
			for account, i := range identity.GetMap() {
				fmt.Printf("ACCOUNT: %s\n%#v\n", account, i)
			}
		case "r":
			fmt.Println(replay.GetCurrentHashForAccount(actors.MyWallet().Account))
		case "c":
			fmt.Println("CURRENT CONFIG")
			for k, v := range actors.MakeOrGetConfig().AllSettings() {
				fmt.Printf("\nKey: %s; Value: %v\n", k, v)
			}
		}
	}
}
