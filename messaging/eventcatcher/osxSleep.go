//go:build darwin

package eventcatcher

import (
	"github.com/prashantgupta24/mac-sleep-notifier/notifier"
)

func sleeper(listen chan bool) {
	sleepNotifier := notifier.GetInstance().Start()
	go func() {
		<-sleepNotifier
		listen <- true
	}()
}
