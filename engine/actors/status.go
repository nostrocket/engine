package actors

import (
	"os"
	"time"

	"github.com/sasha-s/go-deadlock"
)

var terminateChan chan struct{}

func SetTerminateChan(term chan struct{}) {
	terminateChan = term
}

func GetTerminateChan() chan struct{} {
	return terminateChan
}

func Shutdown() {
	close(terminateChan)
	go func() {
		time.Sleep(time.Second * 10)
		os.Exit(1)
	}()
}

var waitGroup *deadlock.WaitGroup

func SetWaitGroup(group *deadlock.WaitGroup) {
	waitGroup = group
}

func GetWaitGroup() *deadlock.WaitGroup {
	return waitGroup
}
