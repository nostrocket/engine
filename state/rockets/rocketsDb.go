package rockets

import (
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/state"
)

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
	var masterocket = state.Rocket{
		RocketUID:  actors.IgnitionRocketID,
		RocketName: "nostrocket",
		CreatedBy:  actors.IgnitionAccount,
		ProblemID:  actors.IgnitionEvent,
	}
	if _, err := state.Upsert(masterocket); err != nil {
		actors.LogCLI(err.Error(), 0)
	}
	close(ready)
	<-actors.GetTerminateChan()
	actors.GetWaitGroup().Done()
	actors.LogCLI("Rockets Mind has shut down", 4)
}

func findRocketByProblemUID(problemUID string) (state.Rocket, bool) {
	for _, rocket := range state.Rockets() {
		if rocket.ProblemID == problemUID {
			return rocket, true
		}
	}
	return state.Rocket{}, false
}

func findRocketByName(name string) (state.Rocket, bool) {
	for _, rocket := range state.Rockets() {
		if rocket.RocketName == name {
			return rocket, true
		}
	}
	return state.Rocket{}, false
}
