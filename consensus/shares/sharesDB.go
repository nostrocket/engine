package shares

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/nbd-wtf/go-nostr"
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

var debug = true

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
	//for s, _ := range subrockets.Names() {
	//	c, ok := actors.Open("shares", s)
	//	if ok {
	//		d := currentState[s]
	//		d.restoreFromDisk(c)
	//		currentState[s] = d
	//	}
	//}
	//if _, ok := currentState["nostrocket"]; !ok {
	k640208 := Kind640208{RocketID: "nostrocket"}
	j, err := json.Marshal(k640208)
	if err != nil {
		library.LogCLI(err.Error(), 0)
	}
	if _, err := handle640208(nostr.Event{
		PubKey:  actors.IgnitionAccount,
		Content: fmt.Sprintf("%s", j),
	}); err != nil {
		library.LogCLI(err.Error(), 0)
	}
	//}
	if debug {
		fmt.Println(currentState["nostrocket"].data)
		currentState["nostrocket"].data["7543214dd1afe9b89d9bcd9d3b64d4596b9bdeb9385e95dabc242608de401099"] = Share{
			LeadTimeLockedShares:   10,
			LeadTime:               1,
			LastLtChange:           0,
			LeadTimeUnlockedShares: 0,
			OpReturnAddresses:      nil,
		}
	}
	close(ready)
	<-actors.GetTerminateChan()
	//for _, d := range currentState {
	//	d.mutex.Lock()
	//	defer d.mutex.Unlock()
	//	d.persistToDisk()
	//}
	actors.GetWaitGroup().Done()
	library.LogCLI("Shares Mind has shut down", 4)
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

//// persistToDisk persists the current state to disk
//func (s *db) persistToDisk() {
//	b, err := json.MarshalIndent(s.data, "", " ")
//	if err != nil {
//		library.LogCLI(err.Error(), 0)
//	}
//	actors.Write("shares", s.rocketID, b)
//}

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

func VotepowerForAccount(account library.Account) int64 {
	startDb()
	currentState["nostrocket"].mutex.Lock()
	defer currentState["nostrocket"].mutex.Unlock()
	if shares, ok := currentState["nostrocket"].data[account]; ok {
		return shares.LeadTime * shares.LeadTimeLockedShares
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
			total = total + (share.LeadTimeLockedShares * share.LeadTime)
		}
		if total > (9223372036854775807 / 5) {
			library.LogCLI("we are 20% of the way to an overflow bug", 1)
		}
		if total > (9223372036854775807 / 2) {
			return 0, fmt.Errorf("we are 50%% of the way to an overflow bug")
		}
		return total, nil
	}
	return 0, fmt.Errorf("no nostrocket state in shares mind")
}

func Permille(signed, total int64) (int64, error) {
	if signed > total || total == 0 {
		return 0, fmt.Errorf("invalid permille, numerator %d is greater than denominator %d", signed, total)
	}
	s := new(big.Rat)
	fmt.Printf("signed: %d total: %d\n", signed, total)
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
	var shares []struct {
		acc       library.Account
		votepower int64
	}
	m := getMapped()
	if d, ok := m["nostrocket"]; ok {
		for l, share := range d {
			shares = append(shares, struct {
				acc       library.Account
				votepower int64
			}{acc: l, votepower: share.LeadTime * share.LeadTimeLockedShares})
		}
	}
	sort.Slice(shares, func(i, j int) bool {
		if shares[i].votepower > shares[j].votepower {
			return true
		}
		return false
	})
	for i, share := range shares {
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
