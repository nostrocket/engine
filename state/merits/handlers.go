package merits

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
	"nostrocket/state/identity"
	"nostrocket/state/problems"
	"nostrocket/state/rockets"
)

func HandleEvent(event nostr.Event) (m Mapped, err error) {
	startDb()
	if !identity.IsUSH(event.PubKey) {
		return nil, fmt.Errorf("event %s: pubkey %s not in identity tree", event.ID, event.PubKey)
	}
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	switch event.Kind {
	case 1:
		return handleByTags(event)
	default:
		return nil, fmt.Errorf("I am the merits mind, event %s was sent to me but I don't know how to handle kind %d", event.ID, event.Kind)
	}
}

func handleByTags(event nostr.Event) (m Mapped, e error) {
	if operation, ok := library.GetFirstTag(event, "op"); ok {
		ops := strings.Split(operation, ".")
		if len(ops) > 2 {
			if ops[1] == "merits" {
				switch o := ops[2]; {
				case o == "register":
					return handleCreateNewCapTable(event)
				case o == "newrequest":
					return handleNewMeritRequest(event)
				}
			}
		}
	}
	return nil, fmt.Errorf("no valid operation found 35645ft")
}

func handleNewMeritRequest(event nostr.Event) (m Mapped, e error) {
	rocketID, ok := library.GetOpData(event, "rocket")
	if !ok {
		return nil, fmt.Errorf("%s tried to create a new expense request but no rocket was specified", event.ID)
	}
	existingRocketData, ok := currentState[rocketID]
	if !ok {
		return nil, fmt.Errorf("%s tried to create a new expense request but the rocketID %s was not found", event.ID, rocketID)
	}
	existingMeritData, ok := existingRocketData.data[event.PubKey]
	if !ok {
		existingMeritData = Merit{
			RocketID:               rocketID,
			LeadTimeLockedMerits:   0,
			LeadTime:               0,
			LastLtChange:           0, //todo use current bitcoin height
			LeadTimeUnlockedMerits: 0,
			OpReturnAddresses:      []string{},
			Requests:               []Request{},
		}
	}
	if existingMeritData.RocketID != rocketID {
		return nil, fmt.Errorf("bug k454k9dr3")
	}
	problemID, ok := library.GetOpData(event, "problem")
	if !ok {
		return nil, fmt.Errorf("%s tried to create a new expense request but no problem ID was specified", event.ID)
	}
	amountStr, ok := library.GetOpData(event, "amount")
	if !ok {
		return nil, fmt.Errorf("%s tried to create a new expense request but no amount was specified", event.ID)
	}
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s tried to create a new expense request but error occured when parsing integer from string: %s", event.ID, err.Error())
	}
	if amount < 1 {
		return nil, fmt.Errorf("%s tried to create a new expense request but amount is less than 1", event.ID)
	}
	problem, ok := problems.GetMap()[problemID]
	if !ok {
		return nil, fmt.Errorf("%s tried to create a new expense request but problem ID %s was not found", event.ID, problemID)
	}
	if !problem.Closed {
		return nil, fmt.Errorf("%s tried to create a new expense request but the problem specified in the request is still open", event.ID)
	}
	if existingWithThisProblem, exists := findMeritRequestByProblemID(problemID); exists {
		return nil, fmt.Errorf("%s tried to create a new expense request but the problem specified has been used in merit request %s", event.ID, existingWithThisProblem.UID)
	}
	var request = Request{
		CreatedBy:    event.PubKey,
		RocketID:     rocketID,
		UID:          event.ID,
		Problem:      problemID,
		Amount:       amount,
		RepaidAmount: 0,
		WitnessedAt:  0, //todo add current Bitcoin height
	}
	existingMeritData.Requests = append(existingMeritData.Requests, request)
	currentState[rocketID].data[event.PubKey] = existingMeritData
	return getMapped(), nil
}

func handleCreateNewCapTable(event nostr.Event) (m Mapped, e error) {
	var rocketID string
	var founder library.Account
	var ok bool
	if rocketID, ok = library.GetOpData(event, ""); !ok {
		return nil, fmt.Errorf("no valid operation found 678yug")
	}
	if founder, ok = rockets.RocketCreators()[rocketID]; !ok {
		return m, fmt.Errorf("%s tried to create a new cap table for rocket %s, but the rocket mind reports no such rocket exists", event.ID, rocketID)
	}
	if founder != event.PubKey {
		return m, fmt.Errorf("%s tried to create a new cap table for rocket %s, but the rocket is owned by %s", event.ID, rocketID, founder)
	}
	if err := makeNewCapTable(rocketID); err != nil {
		return m, fmt.Errorf("%s tried to create a new cap table for rocket %s, but %s", event.ID, rocketID, err.Error())
	}
	d := currentState[rocketID]
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.data[event.PubKey] = Merit{
		RocketID:               rocketID,
		LeadTimeLockedMerits:   1,
		LeadTime:               1,
		LastLtChange:           0, //todo current bitcoin height
		LeadTimeUnlockedMerits: 0,
	}
	currentState[rocketID] = d
	return getMapped(), nil

}
