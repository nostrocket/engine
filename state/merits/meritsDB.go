package merits

import (
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/state/rockets"
)

type meritsForRocket struct {
	rocketID library.RocketID
	data     map[library.Account]Merit
	mutex    *deadlock.Mutex
}

var currentState = make(map[library.RocketID]meritsForRocket)
var currentStateMu = &deadlock.Mutex{}

var started = false
var available = &deadlock.Mutex{}

var debug = false

// StartDb starts the database for this mind (the Mind-state). It blocks until the database is ready to use.
func startDb() {
	available.Lock()
	defer available.Unlock()
	if !started {
		started = true
		for s, _ := range rockets.RocketCreators() {
			currentState[s] = meritsForRocket{
				rocketID: s,
				data:     make(map[library.Account]Merit),
				mutex:    &deadlock.Mutex{},
			}
		}

		// we need a channel to listen for a successful database start
		ready := make(chan struct{})
		// now we can start the database in a new goroutine
		go start(ready)
		// when the database has started, the goroutine will close the `ready` channel.
		<-ready //This channel listener blocks until closed by `startDb`.
		actors.LogCLI("Merits Mind has started", 4)
	}
}

func start(ready chan struct{}) {
	// We add a delta to the provided waitgroup so that upstream knows when the database has been safely shut down
	actors.GetWaitGroup().Add(1)
	err := makeNewCapTable(actors.IgnitionRocketID)
	if err != nil {
		actors.LogCLI(err.Error(), 0)
	}
	d := currentState[actors.IgnitionRocketID]
	d.mutex.Lock()

	d.data[actors.IgnitionAccount] = Merit{
		LeadTimeLockedMerits:   1,
		LeadTime:               1,
		LastLtChange:           0,
		LeadTimeUnlockedMerits: 0,
	}
	currentState[actors.IgnitionRocketID] = d
	if debug {
		fmt.Println(currentState[actors.IgnitionRocketID].data)
		currentState[actors.IgnitionRocketID].data["7543214dd1afe9b89d9bcd9d3b64d4596b9bdeb9385e95dabc242608de401099"] = Merit{
			LeadTimeLockedMerits:   10,
			LeadTime:               1,
			LastLtChange:           0,
			LeadTimeUnlockedMerits: 0,
		}
	}
	d.mutex.Unlock()
	close(ready)
	<-actors.GetTerminateChan()
	//for _, d := range currentState {
	//	d.mutex.Lock()
	//	defer d.mutex.Unlock()
	//	d.persistToDisk()
	//}
	actors.GetWaitGroup().Done()
	actors.LogCLI("Merits Mind has shut down", 4)
}

func makeNewCapTable(rocketID library.Sha256) error {
	if table, exists := currentState[rocketID]; exists {
		if len(table.data) > 0 {
			return fmt.Errorf("this cap table already exists")
		}
	}
	currentState[rocketID] = meritsForRocket{
		rocketID: rocketID,
		data:     make(map[library.Account]Merit),
		mutex:    &deadlock.Mutex{},
	}
	return nil
}

func getMapped() Mapped {
	mOuter := make(map[library.RocketID]map[library.Account]Merit)
	for id, d := range currentState {
		mOuter[id] = make(map[library.Account]Merit)
		for account, share := range d.data {
			mOuter[id][account] = share
		}
	}
	return mOuter
}

func VotepowerForAccount(account library.Account) int64 {
	startDb()
	currentState[actors.IgnitionRocketID].mutex.Lock()
	defer currentState[actors.IgnitionRocketID].mutex.Unlock()
	v, _ := computeVotepower(account, actors.IgnitionRocketID)
	return v
}

func GetPermille(account library.Account, rocket library.RocketID) (int64, error) {
	return Permille(computeVotepower(account, rocket))
}

func computeVotepower(account library.Account, rocketID library.RocketID) (a int64, total int64) {
	startDb()
	if len(rocketID) != 64 {
		rocketID = actors.IgnitionRocketID
	}
	if rocketDb, ok := currentState[rocketID]; ok {
		for _, merit := range rocketDb.data {
			total = total + (merit.LeadTimeLockedMerits * merit.LeadTime)
		}
		if total > (9223372036854775807 / 5) {
			actors.LogCLI("we are 20% of the way to an overflow bug", 1)
		}
		if total > (9223372036854775807 / 2) {
			actors.LogCLI("we are 50%% of the way to an overflow bug", 0)
			actors.Shutdown()
		}
		if merits, ok := rocketDb.data[account]; ok {
			return merits.LeadTime * merits.LeadTimeLockedMerits, total
		}
	}
	return 0, total
}

func TotalVotepower() (int64, error) {
	startDb()
	currentState[actors.IgnitionRocketID].mutex.Lock()
	defer currentState[actors.IgnitionRocketID].mutex.Unlock()
	if data, ok := currentState[actors.IgnitionRocketID]; ok {
		var total int64
		for _, share := range data.data {
			total = total + (share.LeadTimeLockedMerits * share.LeadTime)
		}
		if total > (9223372036854775807 / 5) {
			actors.LogCLI("we are 20% of the way to an overflow bug", 1)
		}
		if total > (9223372036854775807 / 2) {
			return 0, fmt.Errorf("we are 50%% of the way to an overflow bug")
		}
		return total, nil
	}
	return 0, fmt.Errorf("no nostrocket state in merits mind")
}

func Permille(signed, total int64) (int64, error) {
	if signed > total || total == 0 {
		return 0, fmt.Errorf("invalid permille, numerator %d is greater than denominator %d", signed, total)
	}
	s := new(big.Rat)
	s = s.SetFrac64(signed, total)
	m := new(big.Rat)
	m.SetInt64(1000)
	s = s.Mul(s, m)
	f, _ := s.Float64()
	return int64(math.Round(f)), nil
}

func GetPosition(account library.Account) int64 {
	startDb()
	currentState[actors.IgnitionRocketID].mutex.Lock()
	defer currentState[actors.IgnitionRocketID].mutex.Unlock()
	return getPostion(account)
}

func getPostion(account library.Account) int64 {
	var merits []struct {
		acc       library.Account
		votepower int64
	}
	m := getMapped()
	if d, ok := m[actors.IgnitionRocketID]; ok {
		for l, merit := range d {
			merits = append(merits, struct {
				acc       library.Account
				votepower int64
			}{acc: l, votepower: merit.LeadTime * merit.LeadTimeLockedMerits})
		}
	}
	sort.Slice(merits, func(i, j int) bool {
		if merits[i].votepower > merits[j].votepower {
			return true
		}
		return false
	})
	for i, share := range merits {
		if share.acc == account {
			return int64(i) + 1
		}
	}
	return 0
}

func GetMapped() Mapped {
	startDb()
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	return getMapped()
}

func findMeritRequestByProblemID(problemID string) (Request, bool) {
	for _, d := range currentState {
		for _, merit := range d.data {
			for _, request := range merit.Requests {
				if request.Problem == problemID {
					return request, true
				}
			}
		}
	}
	return Request{}, false
}

func getNth(rocketID library.RocketID) int64 {
	var latest int64
	for _, merit := range currentState[rocketID].data {
		for _, request := range merit.Requests {
			if request.Nth > latest {
				latest = request.Nth
			}
		}
	}
	return latest + 1
}
