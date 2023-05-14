package replay

import (
	"time"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
)

func HandleEvent(event nostr.Event) (chan bool, chan Mapped, bool) {
	startDb()
	claimedHash, ok := library.GetFirstTag(event, "r")
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
