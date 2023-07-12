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

type db struct {
	rocketID library.RocketName
	data     map[library.Account]Merit
	mutex    *deadlock.Mutex
}

var currentState = make(map[library.RocketName]db)
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
		for s, _ := range rockets.NamesAndFounders() {
			currentState[s] = db{
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
	//k640208 := Kind640208{RocketName: "nostrocket"}
	//j, err := json.Marshal(k640208)
	//if err != nil {
	//	actors.LogCLI(err.Error(), 0)
	//}
	//if _, err := handle640208(nostr.Event{
	//	PubKey:  actors.IgnitionAccount,
	//	Content: fmt.Sprintf("%s", j),
	//}); err != nil {
	//	actors.LogCLI(err.Error(), 0)
	//}
	//}
	err := makeNewCapTable("nostrocket")
	if err != nil {
		actors.LogCLI(err.Error(), 0)
	}
	d := currentState["nostrocket"]
	d.mutex.Lock()

	d.data[actors.IgnitionAccount] = Merit{
		LeadTimeLockedMerits:   1,
		LeadTime:               1,
		LastLtChange:           0,
		LeadTimeUnlockedMerits: 0,
	}
	currentState["nostrocket"] = d
	if debug {
		fmt.Println(currentState["nostrocket"].data)
		currentState["nostrocket"].data["7543214dd1afe9b89d9bcd9d3b64d4596b9bdeb9385e95dabc242608de401099"] = Merit{
			LeadTimeLockedMerits:   10,
			LeadTime:               1,
			LastLtChange:           0,
			LeadTimeUnlockedMerits: 0,
			OpReturnAddresses:      nil,
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

func makeNewCapTable(name library.RocketName) error {
	if table, exists := currentState[name]; exists {
		if len(table.data) > 0 {
			return fmt.Errorf("this cap table already exists")
		}
	}
	currentState[name] = db{
		rocketID: name,
		data:     make(map[library.Account]Merit),
		mutex:    &deadlock.Mutex{},
	}
	return nil
}

func getMapped() Mapped {
	mOuter := make(map[library.RocketName]map[library.Account]Merit)
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
	currentState["nostrocket"].mutex.Lock()
	defer currentState["nostrocket"].mutex.Unlock()
	if merits, ok := currentState["nostrocket"].data[account]; ok {
		return merits.LeadTime * merits.LeadTimeLockedMerits
	}
	return 0
}

func TotalVotepower() (int64, error) {
	startDb()
	currentState["nostrocket"].mutex.Lock()
	defer currentState["nostrocket"].mutex.Unlock()
	if data, ok := currentState["nostrocket"]; ok {
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
	currentState["nostrocket"].mutex.Lock()
	defer currentState["nostrocket"].mutex.Unlock()
	return getPostion(account)
}

func getPostion(account library.Account) int64 {
	var merits []struct {
		acc       library.Account
		votepower int64
	}
	m := getMapped()
	if d, ok := m["nostrocket"]; ok {
		for l, share := range d {
			merits = append(merits, struct {
				acc       library.Account
				votepower int64
			}{acc: l, votepower: share.LeadTime * share.LeadTimeLockedMerits})
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
	currentState["nostrocket"].mutex.Lock()
	defer currentState["nostrocket"].mutex.Unlock()
	return getMapped()
}
