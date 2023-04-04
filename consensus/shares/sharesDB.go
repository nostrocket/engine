package shares

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sasha-s/go-deadlock"
	"nostrocket/consensus/subrockets"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

type db struct {
	rocketID library.RocketID
	data     map[library.Account]Share
	mutex    *deadlock.Mutex
}

var currentState = make(map[library.RocketID]db)
var currentStateMu = &deadlock.Mutex{}

var started = false
var available = &deadlock.Mutex{}

// StartDb starts the database for this mind (the Mind-state). It blocks until the database is ready to use.
func startDb() {
	available.Lock()
	defer available.Unlock()
	if !started {
		started = true
		for s, _ := range subrockets.Names() {
			currentState[s] = db{
				rocketID: s,
				data:     make(map[library.Account]Share),
				mutex:    &deadlock.Mutex{},
			}
		}

		// we need a channel to listen for a successful database start
		ready := make(chan struct{})
		// now we can start the database in a new goroutine
		go start(ready)
		// when the database has started, the goroutine will close the `ready` channel.
		<-ready //This channel listener blocks until closed by `startDb`.
		library.LogCLI("Shares Mind has started", 4)
	}
}

func start(ready chan struct{}) {
	// We add a delta to the provided waitgroup so that upstream knows when the database has been safely shut down
	actors.GetWaitGroup().Add(1)
	// Load current shares from disk
	for s, _ := range subrockets.Names() {
		c, ok := actors.Open("shares", s)
		if ok {
			d := currentState[s]
			d.restoreFromDisk(c)
			currentState[s] = d
		}
	}
	close(ready)
	<-actors.GetTerminateChan()
	for _, d := range currentState {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		d.persistToDisk()
	}
	actors.GetWaitGroup().Done()
	library.LogCLI("Shares Mind has shut down", 4)
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
	actors.Write("shares", s.rocketID, b)
}

func makeNewCapTable(name library.RocketID) error {
	if table, exists := currentState[name]; exists {
		if len(table.data) > 0 {
			return fmt.Errorf("this cap table already exists")
		}
	}
	currentState[name] = db{
		rocketID: name,
		data:     make(map[library.Account]Share),
		mutex:    &deadlock.Mutex{},
	}
	return nil
}

func getMapped() Mapped {
	mOuter := make(map[library.RocketID]map[library.Account]Share)
	for id, d := range currentState {
		mOuter[id] = make(map[library.Account]Share)
		for account, share := range d.data {
			mOuter[id][account] = share
		}
	}
	return mOuter
}
