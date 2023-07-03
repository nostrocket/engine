package actors

import (
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
}

var waitGroup *deadlock.WaitGroup

func SetWaitGroup(group *deadlock.WaitGroup) {
	waitGroup = group
}

func GetWaitGroup() *deadlock.WaitGroup {
	return waitGroup
}
