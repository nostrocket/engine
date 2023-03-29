package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/viper"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func main() {
	conf := viper.New()
	// Now we initialise this configuration with basic settings that are required on startup.
	actors.InitConfig(conf)
	// make the config accessible globally
	actors.SetConfig(conf)
	sendChan := make(chan nostr.Event)
	startRelays(sendChan)
	sendChan <- createEvent()
	time.Sleep(time.Second * 5)
}

func createEvent() nostr.Event {
	e := nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Now(),
		Kind:      640000,
		Tags: nostr.Tags{nostr.Tag{
			"e", "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c", "", "reply"},
		},
		Content: "Current State",
	}

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
	e.Sign(actors.MyWallet().PrivateKey)
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
