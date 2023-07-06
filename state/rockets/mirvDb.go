package rockets

import (
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

type db struct {
	data  map[library.RocketID]Rocket
	mutex *deadlock.Mutex
}

var currentState = db{
	data:  make(map[library.RocketID]Rocket),
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
		actors.LogCLI("Rockets Mind has started", 4)
	}
}

func start(ready chan struct{}) {
	// We add a delta to the provided waitgroup so that upstream knows when the database has been safely shut down
	actors.GetWaitGroup().Add(1)
	// Load current shares from disk
	//c, ok := actors.Open("mirvs", "current")
	//if ok {
	//	currentState.restoreFromDisk(c)
	//}
	if _, exists := currentState.data["nostrocket"]; !exists {
		currentState.data["nostrocket"] = Rocket{
			RocketID:  "nostrocket",
			CreatedBy: actors.IgnitionAccount,
			ProblemID: actors.IgnitionEvent,
		}
	}
	//currentState.persistToDisk()
	close(ready)
	<-actors.GetTerminateChan()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	//b, err := json.MarshalIndent(currentState.data, "", " ")
	//if err != nil {
	//	library.LogCLI(err.Error(), 0)
	//}
	//actors.Write("mirvs", "current", b)
	//currentState.persistToDisk()
	actors.GetWaitGroup().Done()
	actors.LogCLI("Rockets Mind has shut down", 4)
}

//func (s *db) restoreFromDisk(f *os.File) {
//	s.mutex.Lock()
//	err := json.NewDecoder(f).Decode(&s.data)
//	if err != nil {
//		if err.Error() != "EOF" {
//			library.LogCLI(err.Error(), 0)
//		}
//	}
//	s.mutex.Unlock()
//	err = f.Close()
//	if err != nil {
//		library.LogCLI(err.Error(), 0)
//	}
//}
//
//// persistToDisk persists the current state to disk
//func (s *db) persistToDisk() {
//	b, err := json.MarshalIndent(s.data, "", " ")
//	if err != nil {
//		library.LogCLI(err.Error(), 0)
//	}
//	actors.Write("mirvs", "current", b)
//}

func GetMap() Mapped {
	startDb()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return getMap()
}

func getMap() Mapped {
	m := make(map[library.RocketID]Rocket)
	for key, val := range currentState.data {
		m[key] = val
	}
	return m
}

func (s *db) upsert(key library.RocketID, val Rocket) {
	val.RocketID = key
	s.data[key] = val
}
