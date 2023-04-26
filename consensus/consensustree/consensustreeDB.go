package consensustree

import (
	"encoding/json"
	"os"

	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

type db struct {
	data  map[int64]map[library.Sha256]TreeEvent
	mutex *deadlock.Mutex
}

var currentState = db{
	data:  make(map[int64]map[library.Sha256]TreeEvent),
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
		library.LogCLI("Consensus Tree Mind has started", 4)
	}
}

func start(ready chan struct{}) {
	// We add a delta to the provided waitgroup so that upstream knows when the database has been safely shut down
	actors.GetWaitGroup().Add(1)
	// Load current shares from disk
	c, ok := actors.Open("consensustree", "current")
	if ok {
		currentState.restoreFromDisk(c)
	}
	close(ready)
	<-actors.GetTerminateChan()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	b, err := json.MarshalIndent(currentState.data, "", " ")
	if err != nil {
		library.LogCLI(err.Error(), 0)
	}
	actors.Write("consensustree", "current", b)
	currentState.persistToDisk()
	actors.GetWaitGroup().Done()
	library.LogCLI("Consensus Tree Mind has shut down", 4)
}

func (s *db) restoreFromDisk(f *os.File) {
	s.mutex.Lock()
	err := json.NewDecoder(f).Decode(&s.data)
	if err != nil {
		if err.Error() != "EOF" {
			library.LogCLI(err.Error(), 0)
		}
	}
	s.mutex.Unlock()
	err = f.Close()
	if err != nil {
		library.LogCLI(err.Error(), 0)
	}
}

// persistToDisk persists the current state to disk
func (s *db) persistToDisk() {
	b, err := json.MarshalIndent(s.data, "", " ")
	if err != nil {
		library.LogCLI(err.Error(), 0)
	}
	actors.Write("consensustree", "current", b)
}

//func getMyLastest() (library.Sha256, int64) {
//	var heighest int64
//	var eventID library.Sha256
//	//find the latest stateChangeEvent that we have signed
//	for i, m := range currentState.data {
//		for sha256, event := range m {
//			if event.IHaveSigned {
//				if i >= heighest && !event.IHaveReplaced {
//					eventID = sha256
//					heighest = i
//				}
//			}
//		}
//	}
//	if heighest > 0 && len(eventID) == 64 {
//		return eventID, heighest
//	}
//	return actors.ConsensusTree, 0
//}

func GetLatestHandled() (library.Sha256, int64) {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return getLatestHandled()
}

func getLatestHandled() (library.Sha256, int64) {
	var heighest int64
	var eventID library.Sha256
	//find the latest stateChangeEvent that we have signed
	for i, m := range currentState.data {
		for sha256, event := range m {
			if event.StateChangeEventHandled {
				if i >= heighest {
					eventID = sha256
					heighest = i
				}
			}
		}
	}
	if heighest > 0 && len(eventID) == 64 {
		return eventID, heighest
	}
	return actors.ConsensusTree, 0
}

func GetMap() map[int64]map[library.Sha256]TreeEvent {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return getMap()
}

func getMap() map[int64]map[library.Sha256]TreeEvent {
	return currentState.data
}

func GetAllStateChangeEventsInOrder() []library.Sha256 {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return getAllStateChangeEventsInOrder()
}

func getAllStateChangeEventsInOrder() (r []library.Sha256) {
	_, i := getLatestHandled()
	var x int64
	for x = 0; x <= i; x++ {
		var candidate library.Sha256
		var biggestPermille int64
		for _, t := range currentState.data[x] {
			if t.StateChangeEventHandled {
				if t.Permille > biggestPermille {
					candidate = t.StateChangeEventID
					biggestPermille = t.Permille
				}
			}
		}
		r = append(r, candidate)
	}
	return
}
