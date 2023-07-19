package problems

import (
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/state/identity"
)

func HandleEvent(event nostr.Event) (m Mapped, e error) {
	startDb()
	if sig, _ := event.CheckSignature(); !sig {
		return
	}
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	switch event.Kind {
	case 641800:
		return handleNewAnchor(event)
	case 641802:
		return handleContent(event)
	case 641804:
		return handleMetadata(event)
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
					return handleContent(event)
				case o == "claim" || o == "abandon" || o == "close" || o == "open":
					return handleMetaActions(event, o)
				}
			}
		}
	}
	return nil, fmt.Errorf("no valid operation found 543c2345")
}

func handleMetaActions(event nostr.Event, action string) (m Mapped, e error) {
	if !identity.IsUSH(event.PubKey) {
		return nil, fmt.Errorf("pubkey %s is not in the Identity Tree", event.PubKey)
	}
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

func handleMetadata(event nostr.Event) (m Mapped, e error) {
	if anchor, ok := library.GetFirstReply(event); ok {
		if currentProblem, problemExists := currentState.data[anchor]; problemExists {
			if identity.IsUSH(event.PubKey) {
				var updates int64 = 0
				if claim, ok := library.GetFirstTag(event, "claim"); ok {
					if claim == "claim" {
						if !currentProblem.Closed && len(currentProblem.ClaimedBy) == 0 && !hasOpenChildren(anchor) {
							currentProblem.ClaimedBy = event.PubKey
							//todo add bitcoin height to currentProblem.ClaimedAt
							updates++
						}
					}
					if claim == "abandon" {
						if currentProblem.ClaimedBy == event.PubKey ||
							identity.IsMaintainer(event.PubKey) &&
								len(currentProblem.ClaimedBy) == 64 {
							currentProblem.ClaimedBy = ""
							currentProblem.ClaimedAt = 0
							updates++
						}
					}
				}
				if _close, ok := library.GetFirstTag(event, "close"); ok {
					if currentProblem.CreatedBy == event.PubKey || identity.IsMaintainer(event.PubKey) {
						if _close == "close" {
							if !hasOpenChildren(currentProblem.UID) {
								if !currentProblem.Closed {
									currentProblem.Closed = true
									updates++
								}
							}
						}
						if _close == "open" {
							if currentProblem.Closed {
								currentProblem.Closed = false
								updates++
							}
						}
					}
				}
				if updates > 0 {
					currentState.upsert(currentProblem.UID, currentProblem)
					return getMap(), nil
				}
			}
		}
	}
	return nil, fmt.Errorf("no state changed")
}

func handleContent(event nostr.Event) (m Mapped, e error) {
	var updates int64 = 0
	if anchor, ok := library.GetFirstReply(event); ok {
		if identity.IsUSH(event.PubKey) {
			if currentProblem, problemExists := currentState.data[anchor]; problemExists {
				if currentProblem.CreatedBy == event.PubKey || identity.IsMaintainer(event.PubKey) {
					if len(event.Content) > 0 && event.Content != currentProblem.Body && event.Kind == 641802 {
						currentProblem.Body = event.Content
						updates++
					}
					if description, ok := library.GetFirstTag(event, "description"); ok {
						if currentProblem.Body != description && len(description) > 0 {
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
					if updates > 0 {
						currentState.upsert(currentProblem.UID, currentProblem)
						return getMap(), nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("no state changed")
}

func handleNewAnchor(event nostr.Event) (m Mapped, e error) {
	//fmt.Printf("%#v", event)
	//var updates int64 = 0
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
			if identity.IsUSH(event.PubKey) {
				if parentProblem, parentExists := currentState.data[parent]; parentExists {
					if !parentProblem.Closed {
						if len(parentProblem.ClaimedBy) == 0 || parentProblem.ClaimedBy == event.PubKey {
							return insertProblemAnchor(event, parent)
						}
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

	p := Problem{
		UID:       event.ID,
		Parent:    parent,
		Title:     title,
		Body:      description,
		Closed:    false,
		ClaimedAt: 0,
		ClaimedBy: "",
		CreatedBy: event.PubKey,
	}
	currentState.upsert(p.UID, p)
	return getMap(), nil
}
