package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/viper"
	"nostrocket/engine/actors"
	"nostrocket/engine/helpers"
	"nostrocket/engine/library"
)

func main() {
	conf := viper.New()
	//Now we initialise this configuration with basic settings that are required on startup.
	actors.InitConfig(conf)
	//make the config accessible globally
	actors.SetConfig(conf)
	fmt.Println("Current wallet: " + actors.MyWallet().Account)
	sendChan := make(chan nostr.Event)
	startRelays(sendChan)
	sendChan <- createEvent()
	time.Sleep(time.Second * 5)
	//e := createEvent()
	//fmt.Printf("\n%#v\n", e)
	//fmt.Printf("timestamp: %d", e.CreatedAt.Unix())
}

func createEvent() nostr.Event {
	e := helpers.DeleteEvent("80c8497e2fb475bb9d12fb8f70ac5e6688e6d63ee770a003c4a039be62434f0b", "broke consensus")
	//e := nostr.Event{
	//	PubKey:    actors.MyWallet().Account,
	//	CreatedAt: time.Now(),
	//	Kind:      640000,
	//	Tags: nostr.Tags{nostr.Tag{
	//		"e", "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c", "", "reply"},
	//	},
	//	Content: "Current State",
	//}

	//e := nostr.Event{
	//	PubKey:    actors.MyWallet().Account,
	//	CreatedAt: time.Now(),
	//	Kind:      640000,
	//	Tags: nostr.Tags{nostr.Tag{
	//		"e", "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c", "", "root"},
	//		{"e", "7a22f580d253c4142aa4e6b28d577b2d59fdd30083b0eb27ee76a9bd750bff26", "", "reply"},
	//	},
	//	Content: "Identity",
	//}

	//e := nostr.Event{
	//	PubKey:    actors.MyWallet().Account,
	//	CreatedAt: time.Now(),
	//	Kind:      640000,
	//	Tags: nostr.Tags{nostr.Tag{
	//		"e", "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c", "", "root"},
	//		{"e", "7a22f580d253c4142aa4e6b28d577b2d59fdd30083b0eb27ee76a9bd750bff26", "", "reply"},
	//	},
	//	Content: "Shares",
	//}

	//e := nostr.Event{
	//	PubKey:    actors.MyWallet().Account,
	//	CreatedAt: time.Now(),
	//	Kind:      640000,
	//	Tags: nostr.Tags{nostr.Tag{
	//		"e", "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c", "", "root"},
	//		{"e", "7a22f580d253c4142aa4e6b28d577b2d59fdd30083b0eb27ee76a9bd750bff26", "", "reply"},
	//	},
	//	Content: "Subrockets",
	//}
	//e := nostr.Event{
	//	PubKey:    actors.MyWallet().Account,
	//	CreatedAt: time.Now(),
	//	Kind:      640000,
	//	Tags: nostr.Tags{nostr.Tag{
	//		"e", "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c", "", "root"},
	//	},
	//	Content: "Consensus Tree",
	//}

	//e := nostr.Event{
	//	PubKey:    actors.MyWallet().Account,
	//	CreatedAt: time.Now(),
	//	Kind:      640000,
	//	Tags: nostr.Tags{nostr.Tag{
	//		"e", "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c", "", "root"},
	//		{"e", "7a22f580d253c4142aa4e6b28d577b2d59fdd30083b0eb27ee76a9bd750bff26", "", "reply"},
	//	},
	//	Content: "Replay Prevention",
	//}

	//e := nostr.Event{
	//	PubKey:    actors.MyWallet().Account,
	//	CreatedAt: time.Now(),
	//	Kind:      640400,
	//	Tags: nostr.Tags{nostr.Tag{
	//		"e", actors.IgnitionEvent, "", "root"},
	//		{"e", actors.Identity, "", "reply"},
	//		{"r", "f21d7c25d164d55e676c0374e23ab08b962de753cedbe9069c4cb97b17b2d410", "", "reply"},
	//	},
	//}
	//i := identity.Kind640400{
	//	Name:  "gsovereignty",
	//	About: "Just testing another 1",
	//}
	//c, _ := json.Marshal(i)
	//e.Content = fmt.Sprintf("%s", c)
	e.ID = e.GetID()
	e.Sign("")
	//e.Sign(actors.MyWallet().PrivateKey)
	//e.Sign("")
	_, err := e.CheckSignature()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(e.ID)
	return e
}

func startRelays(sendChan chan nostr.Event) {
	relay, err := nostr.RelayConnect(context.Background(), "wss://nostr.688.org")
	if err != nil {
		panic(err)
	}

	go func() {
		select {
		case e := <-sendChan:
			_, err := relay.Publish(context.Background(), e)
			if err != nil {
				library.LogCLI(err.Error(), 2)
			}
		}
	}()
}

type T struct {
	Content string     `json:"content"`
	Tags    [][]string `json:"tags"`
	Kind    int        `json:"kind"`
	Pubkey  string     `json:"pubkey"`
	Id      string     `json:"id"`
}
