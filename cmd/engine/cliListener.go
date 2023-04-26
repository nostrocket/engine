package main

import (
	"fmt"

	"github.com/eiannone/keyboard"
	"nostrocket/consensus/consensustree"
	"nostrocket/consensus/identity"
	"nostrocket/consensus/replay"
	"nostrocket/consensus/shares"
	"nostrocket/engine/actors"
	"nostrocket/messaging/eventconductor"
)

// cliListener is a cheap and nasty way to speed up development cycles. It listens for keypresses and executes commands.
func cliListener(interrupt chan struct{}) {
	fmt.Println("VIEW CURRENT STATE:\ns: cap table\ni: identity table\nw: current wallet\nc: engine config\nC: state change events\nq: to quit\nSee cliListener.go for more")
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
			fmt.Printf("Current Wallet: \n%s\n", actors.MyWallet().Account)
			fmt.Printf("Current Votepower: \n%#v\n", shares.VotepowerForAccount(actors.MyWallet().Account))
		case "i":
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
		case "C":
			fmt.Println("ALL STATE CHANGE EVENTS IN ORDER THEY WERE HANDLED BY THIS ENGINE:")
			for _, sha256 := range consensustree.GetAllStateChangeEventsInOrder() {
				e := eventconductor.GetEventFromCache(sha256)
				fmt.Printf("\nKind: %d Signed By: %s\nContent: %s\n", e.Kind, e.PubKey, e.Content)
			}
			fmt.Println("LATEST STATE CHANGE EVENT AND HEIGHT IN THE CONSENSUS TREE")
			fmt.Println(consensustree.GetLatestHandled())

			//for _, m := range consensustree.GetMap() {
			//	for _, event := range m {
			//		fmt.Printf("\n%#v\n", event)
			//	}
			//}
		}
	}
}
