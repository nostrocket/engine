package problems

import (
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/state"
	"nostrocket/state/identity"
)

func HandleEvent(event nostr.Event) (m Mapped, e error) {
	startDb()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	if !identity.IsUSH(event.PubKey) {
		return nil, fmt.Errorf("event %s is not signed by someone in the identity tree", event.ID)
	}
	switch event.Kind {
	case 1:
		return handleByTags(event)
	}
	return nil, fmt.Errorf("no state changed")
}

func handleByTags(event nostr.Event) (m Mapped, e error) {
	if operation, ok := library.GetFirstTag(event, "op"); ok {
		ops := strings.Split(operation, ".")
		if len(ops) > 2 {
			if ops[1] == "problem" {
				switch o := ops[2]; {
				case o == "create":
					return handleNewAnchor(event)
					//return handleCreationEvent(event)
				case o == "modify":
					return handleModification(event)
				case o == "claim" || o == "abandon" || o == "close" || o == "open":
					return handleMetaActions(event, o)
				}
			}
		}
	}
	return nil, fmt.Errorf("no valid operation found 543c2345")
}

func handleMetaActions(event nostr.Event, action string) (m Mapped, e error) {
	var currentProblem Problem
	currentProblemID, ok := library.GetOpData(event, "")
	if !ok {
		var problemIDsFoundInTags []string
		for _, s := range library.GetAllReplies(event) {
			if problem, ok := currentState.data[s]; ok {
				problemIDsFoundInTags = append(problemIDsFoundInTags, s)
				currentProblem = problem
			}
		}
		if len(problemIDsFoundInTags) != 1 {
			return nil, fmt.Errorf("exactly one problem must be tagged but event %s has tagged %d problem(s)", event.ID, len(problemIDsFoundInTags))
		}
	}
	if ok {
		if problem, ok := currentState.data[currentProblemID]; ok {
			currentProblem = problem
		} else {
			return nil, fmt.Errorf("exactly one valid problem must be tagged but event %s has tagged none", event.ID)
		}
	}
	var updates = 0
	switch action {
	case "claim":
		if hasOpenChildren(currentProblem.UID) {
			return nil, fmt.Errorf("cannot claim a problem that has open children, event ID %s", event.ID)
		}
		if !currentProblem.Closed && len(currentProblem.ClaimedBy) == 0 {
			currentProblem.ClaimedBy = event.PubKey
			//todo add bitcoin height to currentProblem.ClaimedAt
			updates++
		}
	case "abandon":
		if len(currentProblem.ClaimedBy) != 64 {
			return nil, fmt.Errorf("cannot abandon a problem that has not been claimed, event ID %s", event.ID)
		}
		if currentProblem.ClaimedBy != event.PubKey && !identity.IsMaintainer(event.PubKey) {
			return nil, fmt.Errorf("cannot abandon a problem unless signed by problem creator or a maintainer, event ID %s", event.ID)
		}
		currentProblem.ClaimedBy = ""
		currentProblem.ClaimedAt = 0
		updates++
	case "close":
		if hasOpenChildren(currentProblem.UID) {
			return nil, fmt.Errorf("cannot close a problem that has open children, event ID %s", event.ID)
		}
		if !identity.IsMaintainer(event.PubKey) && currentProblem.CreatedBy != event.PubKey {
			return nil, fmt.Errorf("cannot close a problem unless signed by problem creator or a maintainer, event ID %s", event.ID)
		}
		if currentProblem.Closed {
			return nil, fmt.Errorf("cannot close a problem that is already closed, event ID %s", event.ID)
		}
		currentProblem.Closed = true
		updates++
	case "open":
		if currentProblem.CreatedBy != event.PubKey && !identity.IsMaintainer(event.PubKey) {
			return nil, fmt.Errorf("cannot re-open a closed problem unless signed by problem creator or a maintainer, event ID %s", event.ID)
		}
		if !currentProblem.Closed {
			return nil, fmt.Errorf("cannot re-open a problem that is not closed, event ID %s", event.ID)
		}
		currentProblem.Closed = false
		updates++
	default:
		return nil, fmt.Errorf("invalid operation on event %s", event.ID)
	}
	if updates == 0 {
		return nil, fmt.Errorf("event %s did not cause a state change 7y894j5j", event.ID)
	}
	currentState.upsert(currentProblem.UID, currentProblem)
	return getMap(), nil
}

func handleModification(event nostr.Event) (m Mapped, e error) {
	var updates int64 = 0
	var currentProblem Problem
	var found = false
	if anchor, ok := library.GetFirstReply(event); ok {
		if currentP, problemExists := currentState.data[anchor]; problemExists {
			currentProblem = currentP
			found = true
		}
	}
	if anchor, ok := library.GetOpData(event, "target"); ok {
		if currentP, problemExists := currentState.data[anchor]; problemExists {
			currentProblem = currentP
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("could not find a target problem ID in event %s", event.ID)
	}

	if !(currentProblem.CreatedBy == event.PubKey || identity.IsMaintainer(event.PubKey)) {
		return nil, fmt.Errorf("pubkey not authorised to modify problem ID in event %s", event.ID)
	}
	if len(event.Content) > 0 && event.Content != currentProblem.Body && event.Kind == 641802 {
		currentProblem.Body = event.Content
		updates++
	}
	if description, ok := library.GetFirstTag(event, "description"); ok {
		if currentProblem.Body != description && len(description) > 0 {
			currentProblem.Body = description
			currentProblem.Body = description
			updates++
		}
	}
	if title, ok := library.GetFirstTag(event, "title"); ok {
		if currentProblem.Title != title && len(title) > 0 {
			currentProblem.Title = title
			updates++
		}
	}
	if data, exists := library.GetOpData(event, "tag"); exists {
		if len(data) == 64 {
			if rocket, ok := state.Rockets()[data]; ok {
				if rocket.CreatedBy == event.PubKey || data == currentState.data[currentProblem.Parent].Rocket {
					currentProblem.Rocket = data
					updates++
				}
			} else {
				currentProblem.Tags[data] = "" //todo query tags and insert tag name
				updates++
			}
		}
	}
	if updates > 0 {
		currentState.upsert(currentProblem.UID, currentProblem)
		return getMap(), nil
	}
	return nil, fmt.Errorf("no state changed")
}

func handleNewAnchor(event nostr.Event) (m Mapped, e error) {
	if parent, ok := library.GetFirstReply(event); ok {
		//exception for ignition problem
		if len(currentState.data) == 0 && event.PubKey == actors.IgnitionAccount && parent == actors.Problems {
			return insertProblemAnchor(event, actors.Problems)
		}
		//exception for refactor to kind 1
		if event.PubKey == actors.IgnitionAccount && parent == actors.Problems1 {
			currentState.data = make(map[library.Sha256]Problem)
			return insertProblemAnchor(event, actors.Problems1)
		}
		if _, exists := currentState.data[event.ID]; !exists {
			if parentProblem, parentExists := currentState.data[parent]; parentExists {
				if !parentProblem.Closed {
					if len(parentProblem.ClaimedBy) == 0 || parentProblem.ClaimedBy == event.PubKey {
						return insertProblemAnchor(event, parent)
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("no state changed")
}

func insertProblemAnchor(event nostr.Event, parent library.Sha256) (m Mapped, e error) {
	var title string
	var description string
	if d, ok := library.GetFirstTag(event, "description"); ok {
		if len(d) > 0 {
			description = d
		}
	}
	if t, ok := library.GetFirstTag(event, "title"); ok {
		if len(t) > 0 {
			title = t
		}
	}
	if len(title) == 0 && len(event.Content) <= 100 {
		title = event.Content
	}
	rocket := currentState.data[parent].Rocket
	if len(rocket) != 64 {
		rocket = actors.IgnitionRocketID
	}
	p := Problem{
		UID:       event.ID,
		Parent:    parent,
		Title:     title,
		Body:      description,
		Closed:    false,
		ClaimedAt: 0,
		ClaimedBy: "",
		CreatedBy: event.PubKey,
		Tags:      make(map[library.Sha256]string),
		Rocket:    rocket,
	}
	currentState.upsert(p.UID, p)
	return getMap(), nil
}
