package main

import (
	"fmt"
	"time"

	"github.com/eiannone/keyboard"
	"nostrocket/engine/actors"
	"nostrocket/messaging/eventconductor"
	"nostrocket/state/consensustree"
	"nostrocket/state/identity"
	"nostrocket/state/replay"
	"nostrocket/state/shares"
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
		case "s":
			m := shares.GetMapped()
			for id, m2 := range m {
				fmt.Printf("\n--------- Mirv Name: %s -----------\n", id)
				vp, _ := shares.TotalVotepower()
				fmt.Printf("\nTotal Votepower: %d\n", vp)
				for account, share := range m2 {
					pm, _ := shares.Permille(shares.VotepowerForAccount(account), vp)
					fmt.Printf("\nAccount: %s\nLeadTimeLockedShares: %d\nLeadTime: %d\nLastLeadTimeChange: %d\nLeadTimeUnlockedShares: %d\nOpReturnAddresses: %s\nVotepower: %d Permille: %d\n",
						account, share.LeadTimeLockedShares, share.LeadTime, share.LeadTimeUnlockedShares, share.LastLtChange, share.OpReturnAddresses, shares.VotepowerForAccount(account), pm)
				}
				fmt.Printf("\n--------- End of data for: %s -----------\n\n", id)
			}
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
				fmt.Printf("\nID: %s Kind: %d Signed By: %s\nTags: %#v\nContent: %s\n", e.ID, e.Kind, e.PubKey, e.Tags, e.Content)
			}

			fmt.Println("CHECKPOINTS")
			for _, checkpoint := range consensustree.GetCheckpoints() {
				fmt.Printf("\nHeight: %d EventID: %s\nWitnessed At: %s\n", checkpoint.StateChangeEventHeight, checkpoint.StateChangeEventID, time.Unix(checkpoint.CreatedAt, 0).String())
			}
			//fmt.Println("LATEST STATE CHANGE EVENT AND HEIGHT IN THE CONSENSUS TREE")
			//fmt.Println(consensustree.GetLatestHandled())

			for _, m := range consensustree.GetMap() {
				for _, event := range m {
					fmt.Printf("\n%#v\n", event)
				}
			}
		}
	}
}
