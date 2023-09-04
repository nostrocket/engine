package actors

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

func StartRelaysForPublishing(relays []string) chan nostr.Event {
	sendChan := make(chan nostr.Event)
	var chans []chan nostr.Event
	for _, s := range relays {
		chans = append(chans, make(chan nostr.Event))
		relay, err := nostr.RelayConnect(context.Background(), s)
		if err != nil {
			panic(err)
		}
		go func() {
			for {
				select {
				case e := <-chans[len(chans)-1]:
					_, err := relay.Publish(context.Background(), e)
					if err != nil {
						LogCLI(err.Error(), 2)
						//if errchan, ok := e.GetExtra("errors").(chan error); ok {
						//	errchan <- err
						//	continue
						//}
					}
					//if successChan, ok := e.GetExtra("success").(chan bool); ok {
					//	successChan <- true
					//}
				}
			}
		}()
	}
	go func() {
		for {
			select {
			case e := <-sendChan:
				for _, eventChan := range chans {
					//go func(e nostr.Event, eventChan chan nostr.Event) {
					eventChan <- e
					//}(e, eventChan)
				}
			}
		}
	}()
	return sendChan
}
