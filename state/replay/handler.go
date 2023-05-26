package replay

import (
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
)

func HandleEvent(event nostr.Event) (chan bool, chan Mapped, bool) {
	startDb()
	claimedHash, ok := library.GetFirstTag(event, "r")
	if claimedHash == "a9903e3be5376a3d8021dda60fba7bf5f1705f03c5a0eb00ac082226019d710d" {
		fmt.Println(15)
	}
	if ok {
		if claimedHash == getCurrentHashForAccount(event.PubKey) {
			closer := make(chan bool)
			returner := make(chan Mapped)
			go func(chan bool, chan Mapped) {
				select {
				case c := <-closer:
					close(closer)
					if c {
						currentState.mutex.Lock()
						defer currentState.mutex.Unlock()
						currentState.upsert(event.PubKey, event.ID)
						//currentState.persistToDisk()
						returner <- getMap()
					}
					break
				case <-time.After(time.Second * 30):
					close(closer)
					close(returner)
					break
				}
			}(closer, returner)
			return closer, returner, true
		}
	}
	return make(chan bool), make(chan Mapped), false
}
