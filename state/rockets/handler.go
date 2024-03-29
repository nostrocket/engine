package rockets

//if votepower indicates that this problem should not be included in this rocket, then a new rocket MAY be created with this problem
//problem creator can always link the problem to a new rocket even if it falls under an existing rocket

import (
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/library"
	"nostrocket/state"
	"nostrocket/state/identity"
	"nostrocket/state/problems"
)

var lock = &deadlock.Mutex{}

func HandleEvent(event nostr.Event) (m Mapped, e error) {
	startDb()
	if !identity.IsUSH(event.PubKey) {
		return nil, fmt.Errorf("event %s: pubkey %s not in identity tree", event.ID, event.PubKey)
	}
	lock.Lock()
	defer lock.Unlock()
	if event.Kind == 1 {
		result, err := handleEventByTags(event)
		if err != nil {
			return nil, err
		}
		m = make(Mapped)
		for sha256, rocket := range result.(map[library.Sha256]state.Rocket) {
			m[sha256] = rocket
		}
		return
	}
	return m, fmt.Errorf("event %s did not cause a state change", event.ID)
}

func handleEventByTags(event nostr.Event) (m any, e error) {
	if operation, ok := library.GetFirstTag(event, "op"); ok {
		ops := strings.Split(operation, ".")
		if len(ops) > 2 {
			if ops[1] == "rockets" {
				switch o := ops[2]; {
				case o == "register":
					return handleNewRocketName(event)
					//case o == "problem":
					//	return handleLinkRocketToProblem(event)
				}
			}
		}
	}
	return nil, fmt.Errorf("no valid operation found 745fdfg")
}

func handleNewRocketName(event nostr.Event) (m any, e error) {
	var rocketName string
	var ok bool
	if rocketName, ok = library.GetOpData(event, ""); !ok {
		return nil, fmt.Errorf("no valid operation found s45454")
	}
	if existing, exists := findRocketByName(rocketName); exists {
		if len(existing.ProblemID) == 64 {
			return nil, fmt.Errorf("event %s requests creation of new rocket \"%s\" "+
				"but this name is already taken and is associated with problem UID %s",
				event.ID, rocketName, existing.ProblemID)
		}
	}
	currentProblems := problems.GetMap()
	var problemID string
	for _, s := range library.GetAllReplies(event) {
		if _, ok := currentProblems[s]; ok {
			problemID = s
		}
	}
	if len(problemID) != 64 {
		return nil, fmt.Errorf("event %s requests linking rocket "+
			"with problem %s, but this problem doesn't exist j0990j09",
			event.ID, problemID)
	}
	if problem, exists := problems.GetMap()[problemID]; !exists {
		return nil, fmt.Errorf("event %s requests linking rocket "+
			"with problem %s, but this problem doesn't exist",
			event.ID, problemID)
	} else if problem.CreatedBy != event.PubKey {
		return nil, fmt.Errorf("event %s created by %s requests linking rocket "+
			"with problem %s created by %s, problem must be logged by same person who creates the rocket",
			event.ID, event.PubKey, problemID, problem.CreatedBy)
	}
	if len(problemID) != 64 {
		return nil, fmt.Errorf("event %s requests creation of new rocket \"%s\" "+
			"but I could not find the tag of a valid problem UID in the event",
			event.ID, rocketName)
	}
	if problemAdoptedByRocket, exists := findRocketByProblemUID(problemID); exists {
		return nil, fmt.Errorf("event %s requests creation of new rocket \"%s\" "+
			"but the problem %s has already been adopted by Rocket %s",
			event.ID, rocketName, problemID, problemAdoptedByRocket.RocketName)
	}
	return state.Upsert(state.Rocket{
		ProblemID:   problemID,
		RocketUID:   event.ID,
		RocketName:  rocketName,
		CreatedBy:   event.PubKey,
		Maintainers: []library.Account{event.PubKey},
	})
}

//
//func handleLinkRocketToProblem(event nostr.Event) (m Mapped, e error) {
//	var problemUID string
//	var ok bool
//	var existingRocket state.Rocket
//	for _, s := range library.GetAllReplies(event) {
//		if existing, exists := currentState.data[s]; exists {
//			existingRocket = existing
//			ok = true
//		}
//	}
//	if !ok {
//		return nil, fmt.Errorf("event %s refers to a rocket which doesn't exist", event.ID)
//	}
//	if existingRocket.CreatedBy != event.PubKey {
//		return nil, fmt.Errorf("event %s refers to a rocket which is not owned by the same pubkey as this event", event.ID)
//	}
//	if problemUID, ok = library.GetOpData(event, ""); !ok {
//		return nil, fmt.Errorf("no valid operation found 645rty")
//	}
//	if problem, exists := problems.GetMap()[problemUID]; !exists {
//		return nil, fmt.Errorf("event %s requests linking rocket "+
//			"with problem %s, but this problem doesn't exist",
//			event.ID, problemUID)
//	} else if problem.CreatedBy != event.PubKey {
//		return nil, fmt.Errorf("event %s requests linking rocket "+
//			"with problem %s, but this problem was created by someone else",
//			event.ID, problemUID)
//	}
//	if r, exists := findRocketByProblemUID(problemUID); exists {
//		return nil, fmt.Errorf("event %s requests linking rocket "+
//			"with problem %s, but this problem is already linked to rocket \"%s\"",
//			event.ID, problemUID, r.RocketName)
//	}
//	existingRocket.ProblemID = problemUID
//	currentState.upsert(existingRocket.RocketUID, existingRocket)
//	return getMap(), nil
//}
