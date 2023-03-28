package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/viper"
	"nostrocket/consensus/identity"
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
	//e := nostr.Event{
	//	PubKey:    actors.MyWallet().Account,
	//	CreatedAt: time.Now(),
	//	Kind:      640000,
	//	Tags: nostr.Tags{nostr.Tag{
	//		"e", "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c", "", "reply"},
	//	},
	//	Content: "State Change Requests",
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
	//	Content: "Replay Prevention",
	//}

	e := nostr.Event{
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Now(),
		Kind:      640400,
		Tags: nostr.Tags{nostr.Tag{
			"e", "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c", "", "root"},
			{"e", "0a73208becd0b1a9d294e6caef14352047ab44b848930e6979937fe09effaf71", "", "reply"},
			{"e", "e7d743287254f99f3f85db26d0d283800b2d95dd614a93f7a36a698aed284947", "", "reply"},
		},
	}
	i := identity.Kind640400{
		Name:  "gsovereignty2",
		About: "This one MUST fail because name changed",
	}
	c, _ := json.Marshal(i)
	e.Content = fmt.Sprintf("%s", c)
	e.ID = e.GetID()
	e.Sign(actors.MyWallet().PrivateKey)
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
