package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/viper"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func main() {
	const broadcast = true
	conf := viper.New()
	//Now we initialise this configuration with basic settings that are required on startup.
	actors.InitConfig(conf)
	//make the config accessible globally
	actors.SetConfig(conf)
	fmt.Println("Current wallet: " + actors.MyWallet().Account)

	for _, e := range createEvent() {
		e.ID = e.GetID()
		//e.Sign("")
		e.Sign(actors.MyWallet().PrivateKey)
		fmt.Printf("\n%#v\n%d\n", e, e.CreatedAt.Unix())

		sigok, err := e.CheckSignature()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		if !sigok {
			fmt.Println("sig failed")
			return
		}
		if sigok {
			fmt.Println("sig ok")
			if broadcast {
				sendChan := make(chan nostr.Event)
				startRelays(sendChan)
				sendChan <- e
				time.Sleep(time.Second * 15)
			}
		}
	}

	//for {
	//	e := mineEvent()
	//	fmt.Printf("\n%#v\n", e)
	//	fmt.Printf("timestamp: %d", e.CreatedAt.Unix())
	//}
}

func createEvent() (events []nostr.Event) {
	//events = append(events, nostr.Event{
	//	ID:        "a4ec471a326a2dabe30fbf7dfb659794500872595ff4e58d9d75f0fdb5ded18e",
	//	PubKey:    "546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882",
	//	CreatedAt: time.Unix(1687857701, 0),
	//	Kind:      640400,
	//	Tags:      nostr.Tags{nostr.Tag{"nonce", "The Times 23/Jun/2023 Mortgage misery for millions."}},
	//	Content:   fmt.Sprintf("%s", "{\"name\":\"wuwei\",\"about\":\"\"}"), //'{"name":"wuwei","about":""}',
	//	Sig:       "feea4a8b3c770e21008d800626e2252f304c92a7bcd7890c871e787f14ab3c4815f2579577711c16fb011d1ded0f3eddd53f9b171f02d2994ce1ba06614a3fd2",
	//})

	//PROBLEM 0
	events = append(events, nostr.Event{
		ID:        "339d1188c9076d4c44119fca7f29b9b4c32b775853290075e4519ecdfdea4f38",
		Sig:       "d2af8af9d98a38001970b19d39849476de66ec3ed82ec19bb3b908c8fca9a396d4a6c98ce92715398d3827bccf391df959d2090ff1fd892d24ec0cc704b39bc6",
		PubKey:    actors.MyWallet().Account,
		Kind:      641800,
		CreatedAt: time.Unix(1687777777, 0),
		Tags: nostr.Tags{
			nostr.Tag{"e", "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8", "", "root"},
			nostr.Tag{"e", "edea7c22992a1001de805f690d6198fd365ec45e7e5444482100e22447c657a0", "", "reply"},
			nostr.Tag{"r", "9dba1bd226eb668b2f09d8b1a3b195a4f328ac7a33d38c161196c24fd2a6541e"},
		},
		Content: "Problem: we are not living up to our full potential as humanity",
	})

	//IGNITION EVENT note1r0cketrztzx06l3uxd4c2j86fxsfvfls8klsd3aylm38hsqewtyqyp7wj7
	events = append(events, nostr.Event{
		ID:        "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8",
		PubKey:    "546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882",
		CreatedAt: time.Unix(1687497826, 0),
		Kind:      1,
		Tags:      nostr.Tags{nostr.Tag{"nonce", "The Times 23/Jun/2023 Mortgage misery for millions."}},
		Content:   "IGNITION!\n6ae8f23655a19588ebb025af759670cffe2543f4c21b6576f27ceb43bc6873fa",
		Sig:       "eb90eaa1f3c624871a91be7f61c847d208ac9e9b80044f449f314a44defc8e856454d88e5a02c4fa502c585e477456cc2f08d651a90b26913af3e792c9e7bd06",
	})

	//STATE CHANGE REQUESTS
	events = append(events, nostr.Event{
		ID:        "120205879a8d9a38adcb794f7cbff3872c4117a7bb7e86672484f6dee7d6b1c6",
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Unix(1687769789, 0),
		Kind:      640000,
		Tags: nostr.Tags{
			nostr.Tag{"e", "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8", "", "reply"},
		},
		Content: "State Change Requests",
		Sig:     "c5f93ec68cd09fd4c57ea75bccb15ad06c8630ee23a7a46f4fd9e634d3f98b178b4f52d81443ec1ad9ca0ff7bc654ea4c292903b168f1c8c02c0eb2ec0a56a63",
	})

	//IDENTITY STATE CHANGE REQUESTS
	events = append(events, nostr.Event{
		ID:        "320c1d0a15bd0d84c3527862ad02d558df3893dfbbc488dcf7530abec25d23bb",
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Unix(1687772423, 0),
		Kind:      640000,
		Tags: nostr.Tags{nostr.Tag{
			"e", "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8", "", "root"},
			{"e", "120205879a8d9a38adcb794f7cbff3872c4117a7bb7e86672484f6dee7d6b1c6", "", "reply"},
		},
		Content: "Identity State Change Requests",
		Sig:     "285982a11128d1bc5e2a7f3ab323ebcb8153459ac7c4c1674e2c0e9f439283f4cbd55bb7c31b79359f1413bb5e6d92984a4f0fd63ed3f102d6ce21956aaa5d00",
	})

	//CAP TABLE STATE CHANGE REQUESTS
	events = append(events, nostr.Event{
		ID:        "083e612017800c276fbbeda8fe3a965daf63bb3030dd0535cfcd7d06afabb870",
		Sig:       "4e33ac09b0cb8ecf1fde75083805b5da374100236ffa1476f6efe58e288f65ef92acac7b8cf422934d09aa6ca00801f3e286be9e449059ced999a4fdd9fb97df",
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Unix(1687772423, 0),
		Kind:      640000,
		Tags: nostr.Tags{nostr.Tag{
			"e", "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8", "", "root"},
			{"e", "120205879a8d9a38adcb794f7cbff3872c4117a7bb7e86672484f6dee7d6b1c6", "", "reply"},
		},
		Content: "Shares and Cap Table State Change Requests",
	})

	//MIRV State Change Requests
	events = append(events, nostr.Event{
		ID:        "0f56599b6530f1ed1c11745b76a0d0fc29934e9a90accce1521f4dfac7a78532",
		Sig:       "7493ebc20dd2c948ecc252daf43b2a1a453e1f9eb7da966d3c51fc7e2af17a79e5beb91bcac3953a9ab021c20bbb740dc7f95ed419a617913094fb9b629114a3",
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Unix(1687772423, 0),
		Kind:      640000,
		Tags: nostr.Tags{nostr.Tag{
			"e", "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8", "", "root"},
			{"e", "120205879a8d9a38adcb794f7cbff3872c4117a7bb7e86672484f6dee7d6b1c6", "", "reply"},
		},
		Content: "MIRV State Change Requests",
	})

	//PROBLEM TRACKER STATE CHANGE REQUESTS
	events = append(events, nostr.Event{
		ID:        "edea7c22992a1001de805f690d6198fd365ec45e7e5444482100e22447c657a0",
		Sig:       "57a2d54da0158d120621f159aef6a9a91077537e282a65392674c22437c2299f3086a9a4083b724b577320da9439b23606c370a4dc26ae28dee65914eed44f44",
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Unix(1687772770, 0),
		Kind:      640000,
		Tags: nostr.Tags{nostr.Tag{
			"e", "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8", "", "root"},
			{"e", "120205879a8d9a38adcb794f7cbff3872c4117a7bb7e86672484f6dee7d6b1c6", "", "reply"},
		},
		Content: "Problem Tracker State Change Requests",
	})

	//CURRENT STATE
	events = append(events, nostr.Event{
		ID:        "fc54dcb214e86ed3049aec2e26199b457866989da0d9acb2bf8313e023344052",
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Unix(1687770620, 0),
		Kind:      640000,
		Tags: nostr.Tags{nostr.Tag{
			"e", "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8", "", "reply"},
		},
		Content: "Current State",
		Sig:     "dfbb3e13dfd54dfaa3ef0e74de7b4f754d70b08901bdc43d24e0cf05de97bb48d925892246894d33fe0f19b8e1c50c8c53bda02c9719104838f4b9c856396f16",
	})

	//REPLAY PREVENTION
	events = append(events, nostr.Event{
		ID:        "e29992d4c7d272dfc274b8a68f735c76dd361a24cc08bdf2ed6fe8808485024c",
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Unix(1687771196, 0),
		Kind:      640000,
		Tags: nostr.Tags{nostr.Tag{
			"e", "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8", "", "root"},
			{"e", "120205879a8d9a38adcb794f7cbff3872c4117a7bb7e86672484f6dee7d6b1c6", "", "reply"},
		},
		Content: "Replay Prevention",
		Sig:     "3717a1470da83b970dfa2672223dc682df1eb3810d126ec51fb820a22801c25176f094b4c53d6d7ae810b5a864777670b1405024e354b8719df400fde3a1f095",
	})

	//CONSENSUS TREE
	events = append(events, nostr.Event{
		ID:        "0e4eb74ff5031663115958e66ba1538cd4eadaf91f6599c0b0795e6b4c7bc9af",
		PubKey:    actors.MyWallet().Account,
		CreatedAt: time.Unix(1687771493, 0),
		Kind:      640000,
		Tags: nostr.Tags{nostr.Tag{
			"e", "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8", "", "reply"},
		},
		Content: "Consensus Tree",
		Sig:     "f17effeb0ff190c98966fe2b2443d0edbd64dd623f0f7cdff759bda5e4b4a7b880fd49fdf131b917d922afd7e9d834306b209f21826391a57c814907262359c5",
	})

	//{"id":"7227dabb075105b1af089d49f20896ce8809f386b9263aa78224e00b630c9622","pubkey":"b4f36e2a63792324a92f3b7d973fcc33eaa7720aaeee71729ac74d7ba7677675","created_at":1684063676,"kind":641800,"tags":[["e","fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c","","root"],["e","7a22f580d253c4142aa4e6b28d577b2d59fdd30083b0eb27ee76a9bd750bff26","","reply"],["r","f5ccd7b56346782d3fb245aa1b5db4c727eae0e317ca569ff852e4eb52051545"]],"content":"Problem: we are not living up to our full potential as humanity","sig":"3ca113ea7233c7e0fa284e85d5a3ab34f6281c35b2dc4c4a81c1b4e28ba40947ff6a062e660ae7277863d9ab98e1a17f08d04c13c1a3aa77711379b7397c9707"}

	//e := helpers.DeleteEvent("80c8497e2fb475bb9d12fb8f70ac5e6688e6d63ee770a003c4a039be62434f0b", "broke consensus")

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
	//e.ID = e.GetID()
	//e.Sign("")
	////e.Sign(actors.MyWallet().PrivateKey)
	////e.Sign("")
	//_, err := e.CheckSignature()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Println(e.ID)
	return
}

func mineEvent() nostr.Event {
	c := make(chan nostr.Event)
	for i := 0; i < 20; i++ {
		go func(c chan nostr.Event) {
			fmt.Println(132)
			for {
				e := nostr.Event{
					PubKey:    actors.MyWallet().Account,
					CreatedAt: time.Now(),
					Kind:      1,
					Tags: nostr.Tags{nostr.Tag{
						"nonce", fmt.Sprintf("%d", rand.Int())},
					},
					Content: "IGNITION!",
				}
				go func(e nostr.Event, c chan nostr.Event) {
					e.ID = e.GetID()
					if strings.HasPrefix(e.ID, "9876543210") {
						e.Sign(actors.MyWallet().PrivateKey)
						_, err := e.CheckSignature()
						if err != nil {
							log.Fatal(err)
						}
						c <- e
					}
					if strings.HasPrefix(e.ID, "876543210") {
						e.Sign(actors.MyWallet().PrivateKey)
						_, err := e.CheckSignature()
						if err != nil {
							log.Fatal(err)
						}
						c <- e
					}
					if strings.HasPrefix(e.ID, "76543210") {
						e.Sign(actors.MyWallet().PrivateKey)
						_, err := e.CheckSignature()
						if err != nil {
							log.Fatal(err)
						}
						c <- e
					}
					if strings.HasPrefix(e.ID, "6543210") {
						e.Sign(actors.MyWallet().PrivateKey)
						_, err := e.CheckSignature()
						if err != nil {
							log.Fatal(err)
						}
						c <- e
					}
				}(e, c)

			}
		}(c)
	}
	return <-c
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

// igniton event
//nostr.Event{ID:"5432100158129320f6337401bd597fbf8964aa9b72994837d145bf27bf13420e", PubKey:"546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882", CreatedAt:time.Date(2023, time.June, 21, 13, 59, 58, 584314000, time.Local), Kind:1, Tags:nostr.Tags{nostr.Tag{"nonce", "5321728510997407547"}}, Content:"IGNITION!", Sig:"89b3edeedd8a4c4b0ee78e0421a38321b607e7c4c88581e1f5305d180882ef87a1b5caa2d2a52bd0178b90f1599227a79d1567273738a38dc4a18ca246007356", extra:map[string]interface {}(nil)}
//timestamp: 1687327198%
