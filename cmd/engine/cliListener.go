package main

import (
	"encoding/json"
	"fmt"

	"github.com/eiannone/keyboard"
	"nostrocket/engine/actors"
	"nostrocket/messaging/blocks"
	"nostrocket/messaging/eventconductor"
	"nostrocket/state/consensustree"
	"nostrocket/state/identity"
	"nostrocket/state/merits"
	"nostrocket/state/problems"
	"nostrocket/state/replay"
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
			m := merits.GetMapped()
			for id, m2 := range m {
				fmt.Printf("\n--------- Rocket Name: %s -----------\n", id)
				vp, _ := merits.TotalVotepower()
				fmt.Printf("\nTotal Votepower: %d\n", vp)
				for account, share := range m2 {
					pm, _ := merits.Permille(merits.VotepowerInNostrocketForAccount(account), vp)
					fmt.Printf("\nAccount: %s\nLeadTimeLockedMerits: %d\nLeadTime: %d\nLastLeadTimeChange: %d\nLeadTimeUnlockedMerits: %d\nVotepower: %d Permille: %d\n",
						account, share.LeadTimeLockedMerits, share.LeadTime, share.LeadTimeUnlockedMerits, share.LastLtChange, merits.VotepowerInNostrocketForAccount(account), pm)
				}
				fmt.Printf("\n--------- End of data for: %s -----------\n\n", id)
			}
		case "q":
			close(interrupt)
		case "w":
			fmt.Printf("Current Wallet: \n%s\n", actors.MyWallet().Account)
			fmt.Printf("Current Votepower: \n%#v\n", merits.VotepowerInNostrocketForAccount(actors.MyWallet().Account))
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
			for i, sha256 := range consensustree.GetAllStateChangeEventsInOrder() {
				e := eventconductor.GetEventFromCache(sha256)
				fmt.Printf("\n%d\nID: %s Kind: %d Signed By: %s\nTags: %#v\nContent: %s\n\n", i, e.ID, e.Kind, e.PubKey, e.Tags, e.Content)
			}

			//fmt.Println("CHECKPOINTS")
			//for _, checkpoint := range consensustree.GetCheckpoints() {
			//	fmt.Printf("\nHeight: %d EventID: %s\nWitnessed At: %s\n", checkpoint.StateChangeEventHeight, checkpoint.StateChangeEventID, time.Unix(checkpoint.CreatedAt, 0).String())
			//}

			//fmt.Println("LATEST STATE CHANGE EVENT AND HEIGHT IN THE CONSENSUS TREE")
			//fmt.Println(consensustree.GetLatestHandled())

			//for _, m := range consensustree.GetMap() {
			//	for _, event := range m {
			//		fmt.Printf("\n%#v\n", event)
			//	}
			//}
		case "p":
			fmt.Println("-------- PROBLEMS ---------")
			for _, problem := range problems.GetMap() {
				fmt.Printf("\nUID: %s\nPARENT: %s\nTITLE: %s\nBODY: %s\nCREATED BY: %s\n\n", problem.UID, problem.Parent, problem.Title, problem.Body, problem.CreatedBy)
			}
		case "f":
			currentState := eventconductor.GetCurrentStateMap()
			if currentState.Identity == nil {
				currentState.Identity = identity.GetMap()
			}
			if currentState.Replay == nil {
				currentState.Replay = replay.GetMap()
			}
			wire := currentState.Wire()
			fmt.Printf("\n%#v\n", wire)
			b, err := json.Marshal(wire)
			if err != nil {
				actors.LogCLI(err, 2)
			} else {
				eventconductor.Publish(eventconductor.CurrentStateEventBuilder(fmt.Sprintf("%s", b)))
			}
		case "o":
			cs := eventconductor.GetCurrentStateMap()
			wire := cs.Wire()
			for id, rocket := range wire.Rockets {
				fmt.Printf("\n%s\n%#v\n", id, rocket.Products)
			}
		case "t":
			t := blocks.Tip()
			fmt.Printf("%#v", t)
			//fmt.Printf("\n%#v\n", wire.Payments.Products)
		}
	}
}
