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
				case o == "vote":
					return handleNewVote(event)
				}
			}
		}
	}
	return nil, fmt.Errorf("no valid operation found 35645ft")
}

func handleNewVote(event nostr.Event) (m Mapped, e error) {
	targetPubkey, ok := library.GetOpData(event, "pubkey")
	if !ok {
		return nil, fmt.Errorf("%s tried to vote on a merit request but did not specify a target pubkey", event.ID)
	}
	existingMeritData, err := getExistingMeritData(event, targetPubkey)
	if err != nil {
		return nil, err
	}
	vpForAccount, totalVotepower := computeVotepower(event.PubKey, existingMeritData.RocketID)
	if vpForAccount == 0 {
		return m, fmt.Errorf("%s tried to vote on a merit request but the pubkey has no votepower in this rocket", event.ID)
	}
	if totalVotepower < vpForAccount {
		return m, fmt.Errorf("%s tried to vote on a merit request but there is a bug affecting votepower", event.ID)
	}
	requestID, ok := library.GetOpData(event, "request")
	if !ok {
		return nil, fmt.Errorf("%s tried to vote on a merit request but could not find a request ID", event.ID)
	}
	direction, ok := library.GetOpData(event, "direction")
	if !ok {
		return nil, fmt.Errorf("%s tried to vote on a merit request but did not specify a voting direction", event.ID)
	}
	for i, request := range existingMeritData.Requests {
		if request.UID == requestID {
			if request.Approved {
				return nil, fmt.Errorf("%s tried to vote on a merit request but this request has already been approved", event.ID)
			}
			if request.Approved {
				return nil, fmt.Errorf("%s tried to vote on a merit request but this request has already been rejected", event.ID)
			}
			if _, exists := request.Ratifiers[event.PubKey]; exists {
				return nil, fmt.Errorf("%s tried to vote on a merit request but this pubkey has already voted on this request", event.ID)
			}
			if _, exists := request.Blackballers[event.PubKey]; exists {
				return nil, fmt.Errorf("%s tried to vote on a merit request but this pubkey has already voted on this request", event.ID)
			}
			if direction == "blackball" {
				existingMeritData.Requests[i].Blackballers[event.PubKey] = struct{}{}
			}
			if direction == "ratify" {
				existingMeritData.Requests[i].Ratifiers[event.PubKey] = struct{}{}
			}
			existingMeritData.Requests[i].BlackballPermille = 0
			for account, _ := range existingMeritData.Requests[i].Blackballers {
				permille, err := GetPermille(account, existingMeritData.RocketID)
				if err != nil {
					return m, err
				}
				existingMeritData.Requests[i].BlackballPermille += permille
			}
			existingMeritData.Requests[i].RatifyPermille = 0
			for account, _ := range existingMeritData.Requests[i].Ratifiers {
				permille, err := GetPermille(account, existingMeritData.RocketID)
				if err != nil {
					return m, err
				}
				existingMeritData.Requests[i].RatifyPermille += permille
			}
			// todo <Rule> An Expense MUST be Approved if it achieves a Ratification Rate of greater than 66.6%, and Blackball Rate of less than 6%, after an Active Period greater than 1,008 Blocks.
			// todo <Rule> An Expense MUST be Approved if it achieves a Ratification Rate of greater than 50%, and Blackball Rate of no greater than 0% after an Active Period greater than 144 Blocks.
			// <Rule> An Expense MUST be Approved if it achieves a Ratification Rate of greater than 90% and a Blackball Rate of no greater than 0% after an Active Period greater than 0 Blocks.
			if existingMeritData.Requests[i].BlackballPermille == 0 && existingMeritData.Requests[i].RatifyPermille > 900 {
				existingMeritData.Requests[i].Approved = true
				existingMeritData.LeadTimeUnlockedMerits += existingMeritData.Requests[i].Amount
				existingMeritData.Requests[i].MeritsCreated = existingMeritData.Requests[i].Amount
				existingMeritData.Requests[i].Nth = getNth(existingMeritData.RocketID)
			}
			// <Rule> An Expense MUST be Rejected if it achieves a Blackball Rate of greater than 20%
			if existingMeritData.Requests[i].BlackballPermille > 200 {
				existingMeritData.Requests[i].Rejected = true
			}
			currentState[existingMeritData.RocketID].data[targetPubkey] = existingMeritData
			return getMapped(), nil
		}
	}
	return nil, fmt.Errorf("unknown error 7y83y824")
}

func getExistingMeritData(event nostr.Event, targetPubkey library.Account) (m Merit, e error) {
	rocketID, ok := library.GetOpData(event, "rocket")
	if !ok {
		return m, fmt.Errorf("%s tried to create a new merit request but no rocket was specified", event.ID)
	}
	existingRocketData, ok := currentState[rocketID]
	if !ok {
		return m, fmt.Errorf("%s tried to create a new merit request but the rocketID %s was not found", event.ID, rocketID)
	}
	existingMeritData, ok := existingRocketData.data[targetPubkey]
	if !ok {
		existingMeritData = Merit{
			RocketID:               rocketID,
			LeadTimeLockedMerits:   0,
			LeadTime:               0,
			LastLtChange:           0, //todo use current bitcoin height
			LeadTimeUnlockedMerits: 0,
			Requests:               []Request{},
		}
	}
	return existingMeritData, nil
}

func handleNewMeritRequest(event nostr.Event) (m Mapped, e error) {
	existingMeritData, err := getExistingMeritData(event, event.PubKey)
	if err != nil {
		return nil, err
	}
	problemID, ok := library.GetOpData(event, "problem")
	if !ok {
		return nil, fmt.Errorf("%s tried to create a new merit request but no problem ID was specified", event.ID)
	}
	amountStr, ok := library.GetOpData(event, "amount")
	if !ok {
		return nil, fmt.Errorf("%s tried to create a new merit request but no amount was specified", event.ID)
	}
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s tried to create a new merit request but error occured when parsing integer from string: %s", event.ID, err.Error())
	}
	if amount < 1 {
		return nil, fmt.Errorf("%s tried to create a new merit request but amount is less than 1", event.ID)
	}
	problem, ok := problems.GetMap()[problemID]
	if !ok {
		return nil, fmt.Errorf("%s tried to create a new merit request but problem ID %s was not found", event.ID, problemID)
	}
	if !problem.Closed {
		return nil, fmt.Errorf("%s tried to create a new merit request but the problem specified in the request is still open", event.ID)
	}
	if existingWithThisProblem, exists := findMeritRequestByProblemID(problemID); exists {
		return nil, fmt.Errorf("%s tried to create a new merit request but the problem specified has been used in merit request %s", event.ID, existingWithThisProblem.UID)
	}
	var request = Request{
		CreatedBy:         event.PubKey,
		OwnedBy:           event.PubKey,
		RocketID:          existingMeritData.RocketID,
		UID:               event.ID,
		Problem:           problemID,
		Amount:            amount,
		RemuneratedAmount: 0,
		WitnessedAt:       0, //todo add current Bitcoin height
		Ratifiers:         make(map[library.Account]struct{}),
		Blackballers:      make(map[library.Account]struct{}),
	}
	existingMeritData.Requests = append(existingMeritData.Requests, request)
	currentState[existingMeritData.RocketID].data[event.PubKey] = existingMeritData
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
