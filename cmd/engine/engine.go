package main

import (
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/viper"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/messaging/eventcatcher"
)

func main() {
	// Various aspect of this application require global and local settings. To keep things
	// clean and tidy we put these settings in a Viper configuration.
	conf := viper.New()

	// Now we initialise this configuration with basic settings that are required on startup.
	actors.InitConfig(conf)
	// make the config accessible globally
	actors.SetConfig(conf)
	fmt.Println("CURRENT CONFIG")
	for k, v := range actors.MakeOrGetConfig().AllSettings() {
		fmt.Printf("\nKey: %s; Value: %v\n", k, v)
	}
	terminateChan := make(chan struct{})
	eventChan := make(chan nostr.Event)
	go eventcatcher.SubscribeToTree(terminateChan, eventChan)
L:
	for {
		select {
		case <-time.After(time.Second * 30):
			close(terminateChan)
			break L
		}
	}
	fmt.Println(library.Bye())
}
