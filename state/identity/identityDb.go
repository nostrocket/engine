package identity

import (
	"encoding/json"
	"net/mail"

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
		actors.LogCLI("Identity Mind has started", 4)
	}
}

func start(ready chan struct{}) {
	// We add a delta to the provided waitgroup so that upstream knows when the database has been safely shut down
	actors.GetWaitGroup().Add(1)
	// Load current state from disk
	//c, ok := actors.Open("identity", "current")
	//if ok {
	//	currentState.restoreFromDisk(c)
	//}
	insertIgnitionState()
	close(ready)
	<-actors.GetTerminateChan()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	//b, err := json.MarshalIndent(currentState.data, "", " ")
	//if err != nil {
	//	library.LogCLI(err.Error(), 0)
	//}
	//actors.Write("identity", "current", b)
	//currentState.persistToDisk()
	actors.GetWaitGroup().Done()
	actors.LogCLI("Identity Mind has shut down", 4)
}

func insertIgnitionState() {
	ignitionAccount := getLatestIdentity(actors.IgnitionAccount)
	if len(ignitionAccount.UniqueSovereignBy) == 0 {
		ignitionAccount.UniqueSovereignBy = "1Humanityrvhus5mFWRRzuJjtAbjk2qwww"
		ignitionAccount.MaintainerBy = "1Humanityrvhus5mFWRRzuJjtAbjk2qwww"
		currentState.upsert(actors.IgnitionAccount, ignitionAccount)
		//currentState.persistToDisk()
	}
}
func getLatestIdentity(account library.Account) Identity {
	id, ok := currentState.data[account]
	if !ok {
		return Identity{}
	}
	return id
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
//	actors.Write("identity", "current", b)
//}

func GetMap() Mapped {
	startDb()
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
	startDb()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	id := getLatestIdentity(account)
	if len(id.MaintainerBy) > 0 {
		return true
	}
	return false
}

func IsUSH(account library.Account) bool {
	startDb()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return isUSH(account)
}

func isUSH(account library.Account) bool {
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

func GetLightningAddress(account library.Account) (string, bool) {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	if data, ok := currentState.data[account]; ok {
		if len(data.LatestKind0.Content) > 0 {
			var profile Profile
			err := json.Unmarshal([]byte(data.LatestKind0.Content), &profile)
			if err == nil {
				addr, err := mail.ParseAddress(profile.Lud16)
				if err == nil {
					return addr.String(), true
				}
			}
		}
	}
	return "", false
}

type Profile struct {
	Name         string `json:"name"`
	Picture      string `json:"picture"`
	About        string `json:"about"`
	Website      string `json:"website"`
	Banner       string `json:"banner"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	DisplayName1 string `json:"displayName"`
	Lud06        string `json:"lud06"`
	Lud16        string `json:"lud16"`
	Nip05        string `json:"nip05"`
	Nip05Valid   bool   `json:"nip05valid"`
}
