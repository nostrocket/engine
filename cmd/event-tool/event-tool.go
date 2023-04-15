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
	e := nostr.Event{
		PubKey:    "d91191e30e00444b942c0e82cad470b32af171764c2275bee0bd99377efd4075",
		CreatedAt: time.Unix(1681362888, 0),
		Kind:      1,
		Tags: nostr.Tags{
			nostr.Tag{
				"e", "8d61f3346a9875bfd135a17793e13b1235843abac2ba86529b58294dadabc23a", "", "reply"},
			nostr.Tag{"p", "d91191e30e00444b942c0e82cad470b32af171764c2275bee0bd99377efd4075"},
		},
		Content: "Anyone who replies in this thread automagically becomes a member of Nostr HK and will get a DM when there's a meetup!",
	}
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
	//e.Sign("")
	fmt.Println(e.ID)
	return e
}

func startRelays(sendChan chan nostr.Event) {
	relay, err := nostr.RelayConnect(context.Background(), "wss://relay.damus.io")
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
