package main

//
//import (
//	"fmt"
//	"time"
//
//	"github.com/nbd-wtf/go-nostr"
//	"github.com/spf13/viper"
//	"nostrocket/engine/actors"
//)
//
//func main() {
//	conf := viper.New()
//	// Now we initialise this configuration with basic settings that are required on startup.
//	actors.InitConfig(conf)
//	// make the config accessible globally
//	actors.SetConfig(conf)
//	handleEvents()
//
//}
//
//func handleEvents() {
//	rxChan := make(chan nostr.Event)
//	txChan := make(chan nostr.Event)
//	eose := make(chan bool)
//	//go eventcatcher.SubscribeToTree(rxChan, txChan, eose)
//	m := make(map[string]nostr.Event)
//	handled := make(map[string]struct{})
//L:
//	for {
//		select {
//		case e := <-rxChan:
//			m[e.ID] = e
//		case <-eose:
//			break L
//		case <-time.After(time.Second * 20):
//			break L
//		}
//	}
//	var mermaid string
//	for _, event := range m {
//		if _, exists := handled[event.ID]; !exists {
//			handled[event.ID] = struct{}{}
//			for _, s2 := range tags(event) {
//				mermaid = mermaid + fmt.Sprintf("\n%s-->%s[%d]", s2, event.ID, event.Kind)
//			}
//		}
//	}
//	fmt.Printf("\n\n%s", mermaid)
//}
//
//func tags(event nostr.Event) []string {
//	var etags []string
//	for _, tag := range event.Tags {
//		if len(tag) >= 2 {
//			if tag[0] == "e" {
//				if len(tag) > 3 {
//					if tag[3] != "root" {
//						etags = append(etags, tag[1])
//					}
//				}
//			}
//		}
//
//	}
//	return etags
//}
