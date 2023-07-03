package replay

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"

	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

type db struct {
	data  map[library.Account]string
	mutex *deadlock.Mutex
}

var currentState = db{
	data:  make(map[library.Account]string),
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
		actors.LogCLI("Replay Mind has started", 4)
	}
}

func start(ready chan struct{}) {
	// We add a delta to the provided waitgroup so that upstream knows when the database has been safely shut down
	actors.GetWaitGroup().Add(1)
	// Load current shares from disk
	//c, ok := actors.Open("replay", "current")
	//if ok {
	//	currentState.restoreFromDisk(c)
	//}
	close(ready)
	<-actors.GetTerminateChan()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	//b, err := json.MarshalIndent(currentState.data, "", " ")
	//if err != nil {
	//	library.LogCLI(err.Error(), 0)
	//}
	//actors.Write("replay", "current", b)
	//currentState.persistToDisk()
	actors.GetWaitGroup().Done()
	actors.LogCLI("Replay Mind has shut down", 4)
}

func (s *db) restoreFromDisk(f *os.File) {
	s.mutex.Lock()
	err := json.NewDecoder(f).Decode(&s.data)
	if err != nil {
		if err.Error() != "EOF" {
			actors.LogCLI(err.Error(), 0)
		}
	}
	s.mutex.Unlock()
	err = f.Close()
	if err != nil {
		actors.LogCLI(err.Error(), 0)
	}
}

//
//// persistToDisk persists the current state to disk
//func (s *db) persistToDisk() {
//	b, err := json.MarshalIndent(s.data, "", " ")
//	if err != nil {
//		library.LogCLI(err.Error(), 0)
//	}
//	actors.Write("replay", "current", b)
//}

func GetMap() Mapped {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return getMap()
}

func getMap() Mapped {
	m := make(map[library.Account]string)
	for account, id := range currentState.data {
		m[account] = id
	}
	return m
}

func (s *db) upsert(account library.Account, last string) {
	s.data[account] = last
}

func GetCurrentHashForAccount(account library.Account) string {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return getCurrentHashForAccount(account)
}

func getCurrentHashForAccount(account library.Account) string {
	if hash, ok := currentState.data[account]; ok {
		return hash
	}
	return actors.ReplayPrevention
}

func GetStateHash() library.Sha256 {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	m := getMap()
	var sl []library.Account
	for account, _ := range m {
		sl = append(sl, account)
	}
	sort.Slice(sl, func(i, j int) bool {
		if sl[i] > sl[j] {
			return true
		}
		return false
	})
	b := bytes.Buffer{}
	for _, account := range sl {
		decodedString, err := hex.DecodeString(m[account])
		if err != nil {
			actors.LogCLI(err, 0)
		}
		b.Write(decodedString)
	}
	return library.Sha256Sum(b.Bytes())
}
