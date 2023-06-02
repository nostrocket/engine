package problems

import (
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

type db struct {
	data  map[library.Sha256]Problem
	mutex *deadlock.Mutex
}

var currentState = db{
	data:  make(map[library.Sha256]Problem),
	mutex: &deadlock.Mutex{},
}

var started = false
var available = &deadlock.Mutex{}

// StartDb starts the database for this mind (the Mind-state). It blocks until the database is ready to use.
func startDb() {
	available.Lock()
	defer available.Unlock()
	if !started {
		started = true
		// we need a channel to listen for a successful database start
		ready := make(chan struct{})
		// now we can start the database in a new goroutine
		go start(ready)
		// when the database has started, the goroutine will close the `ready` channel.
		<-ready //This channel listener blocks until closed by `startDb`.
		library.LogCLI("Problems Mind has started", 4)
	}
}

func start(ready chan struct{}) {
	// We add a delta to the provided waitgroup so that upstream knows when the database has been safely shut down
	actors.GetWaitGroup().Add(1)
	close(ready)
	<-actors.GetTerminateChan()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	actors.GetWaitGroup().Done()
	library.LogCLI("Problems Mind has shut down", 4)
}

func GetMap() Mapped {
	startDb()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return getMap()
}

func getMap() Mapped {
	m := make(map[library.Sha256]Problem)
	for key, val := range currentState.data {
		m[key] = val
	}
	return m
}

func (s *db) upsert(key library.Sha256, val Problem) {
	val.UID = key
	s.data[key] = val
}

func hasOpenChildren(problemID library.Sha256) bool {
	for _, problem := range getMap() {
		if problem.Parent == problemID {
			return true
		}
	}
	return false
}
