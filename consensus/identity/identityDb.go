package identity

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

type db struct {
	data  map[library.Account]Identity
	mutex *deadlock.Mutex
}

var currentState = db{
	data:  make(map[library.Account]Identity),
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
		library.LogCLI("Identity Mind has started", 4)
	}
}

func start(ready chan struct{}) {
	// We add a delta to the provided waitgroup so that upstream knows when the database has been safely shut down
	actors.GetWaitGroup().Add(1)
	// Load current shares from disk
	c, ok := actors.Open("identity", "current")
	if ok {
		currentState.restoreFromDisk(c)
	}
	insertIgnitionState()
	close(ready)
	<-actors.GetTerminateChan()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	b, err := json.MarshalIndent(currentState.data, "", " ")
	if err != nil {
		library.LogCLI(err.Error(), 0)
	}
	actors.Write("identity", "current", b)
	currentState.persistToDisk()
	actors.GetWaitGroup().Done()
	library.LogCLI("Identity Mind has shut down", 4)
}

func insertIgnitionState() {
	fmt.Println(66)
	//todo preflight
	ignitionAccount := getLatestIdentity(actors.IgnitionAccount)
	if len(ignitionAccount.UniqueSovereignBy) == 0 {
		fmt.Println(70)
		ignitionAccount.UniqueSovereignBy = "1Humanityrvhus5mFWRRzuJjtAbjk2qwww"
		ignitionAccount.MaintainerBy = "1Humanityrvhus5mFWRRzuJjtAbjk2qwww"
		currentState.upsert(actors.IgnitionAccount, ignitionAccount)
		currentState.persistToDisk()
		fmt.Println(75)
	}
}
func getLatestIdentity(account library.Account) Identity {
	id, ok := currentState.data[account]
	if !ok {
		return Identity{}
	}
	return id
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
	actors.Write("identity", "current", b)
}

func GetMap() Mapped {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return getMap()
}

func getMap() Mapped {
	m := make(map[library.Account]Identity)
	for account, id := range currentState.data {
		m[account] = id
	}
	return m
}

func IsMaintainer(account library.Account) bool {
	id := getLatestIdentity(account)
	if len(id.MaintainerBy) > 0 {
		return true
	}
	return false
}

func IsUSH(account library.Account) bool {
	id := getLatestIdentity(account)
	if len(id.UniqueSovereignBy) > 0 {
		return true
	}
	return false
}

func (s *db) upsert(account library.Account, identity Identity) {
	identity.Account = account
	s.data[account] = identity
}
