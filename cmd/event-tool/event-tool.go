package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
)

func main() {
	//conf := viper.New()
	// Now we initialise this configuration with basic settings that are required on startup.
	//actors.InitConfig(conf)
	// make the config accessible globally
	//actors.SetConfig(conf)
	//sendChan := make(chan nostr.Event)
	//startRelays(sendChan)
	//sendChan <- createEvent()
	//time.Sleep(time.Second * 5)
	e := createEvent()
	fmt.Printf("\n%#v\n", e)
	fmt.Printf("timestamp: %d", e.CreatedAt.Unix())
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
	//e := nostr.Event{
	//	PubKey:    "d91191e30e00444b942c0e82cad470b32af171764c2275bee0bd99377efd4075",
	//	CreatedAt: time.Unix(1681362888, 0),
	//	Kind:      1,
	//	Tags: nostr.Tags{
	//		nostr.Tag{
	//			"e", "8d61f3346a9875bfd135a17793e13b1235843abac2ba86529b58294dadabc23a", "", "reply"},
	//		nostr.Tag{"p", "d91191e30e00444b942c0e82cad470b32af171764c2275bee0bd99377efd4075"},
	//	},
	//	Content: "Anyone who replies in this thread automagically becomes a member of Nostr HK and will get a DM when there's a meetup!",
	//}

	e := nostr.Event{
		PubKey:    "d91191e30e00444b942c0e82cad470b32af171764c2275bee0bd99377efd4075",
		CreatedAt: time.Now(),
		Kind:      31337,
		Tags: nostr.Tags{
			nostr.Tag{
				"d",
				"b07v7s2ic0haospgmeg73i",
			},
			nostr.Tag{
				"media",
				"https://media.zapstr.live:3118/d91191e30e00444b942c0e82cad470b32af171764c2275bee0bd99377efd4075/naddr1qqtxyvphwcmhxvnfvvcxsct0wdcxwmt9vumnx6gzyrv3ry0rpcqygju59s8g9jk5wzej4ut3wexzyad7uz7ejdm7l4q82qcyqqq856g4xnp7j",
				"http",
			},
			nostr.Tag{
				"p",
				"d91191e30e00444b942c0e82cad470b32af171764c2275bee0bd99377efd4075",
				"Host",
			},
			nostr.Tag{
				"p",
				"fa984bd7dbb282f07e16e7ae87b26a2a7b9b90b7246a44771f0cf5ae58018f52",
				"Guest",
			},
			nostr.Tag{
				"c",
				"Podcast",
			},
			nostr.Tag{
				"price",
				"402",
			},
			nostr.Tag{
				"cover",
				"https://s3-us-west-2.amazonaws.com/anchor-generated-image-bank/production/podcast_uploaded_nologo400/36291377/36291377-1673187804611-64b4f8e9f1687.jpg",
			},
			nostr.Tag{
				"subject",
				"Nostrovia | The Pablo Episode",
			},
		},
		Content: "Nostrovia | The Pablo Episode\n\nhttps://s3-us-west-2.amazonaws.com/anchor-generated-image-bank/production/podcast_uploaded_nologo400/36291377/36291377-1673187804611-64b4f8e9f1687.jpg\n\nhttps://zapstr.live/?track=naddr1qqtxyvphwcmhxvnfvvcxsct0wdcxwmt9vumnx6gzyrv3ry0rpcqygju59s8g9jk5wzej4ut3wexzyad7uz7ejdm7l4q82qcyqqq856g4xnp7j",
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

type T struct {
	Content string     `json:"content"`
	Tags    [][]string `json:"tags"`
	Kind    int        `json:"kind"`
	Pubkey  string     `json:"pubkey"`
	Id      string     `json:"id"`
}
