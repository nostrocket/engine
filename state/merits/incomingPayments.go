package merits

import (
	"fmt"

	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func HandleIncomingPayment(account library.Account, amount int64, rocket library.RocketID, uid library.Sha256) (m Mapped, e error) {
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	if _, exists := handledZaps[uid]; exists {
		return nil, fmt.Errorf("we have aready handled this zap")
	}
	meritsInRocket, ok := currentState[rocket]
	if !ok {
		return nil, fmt.Errorf("could not find merits for this rocket")
	}
	for _, request := range meritsInRocket.orderedRemunerationRequests() {
		if request.OwnedBy == account {
			if err := request.addToRemuneration(amount); err != nil {
				//return nil, err
			} else {
				handledZaps[uid] = struct{}{}
				return getMapped(), nil
			}
		}

	}
	for _, request := range meritsInRocket.orderedDividendRequests() {
		if request.OwnedBy == account {
			if err := request.addToDividends(amount); err != nil {
				return nil, err
			}
			handledZaps[uid] = struct{}{}
			return getMapped(), nil
		}
	}
	actors.LogCLI("this should never happen", 0)
	return nil, fmt.Errorf("this should not happen")
}

func (r Request) addToRemuneration(amount int64) error {
	if err := r.validateExists(); err != nil {
		return err
	}
	if r.Amount-r.RemuneratedAmount <= amount {
		return fmt.Errorf("this would overflow the amount owing to the meritholder")
	}
	for i, request := range currentState[r.RocketID].data[r.OwnedBy].Requests {
		if request.UID == r.UID {
			currentState[r.RocketID].data[r.OwnedBy].Requests[i].RemuneratedAmount += amount
			return nil
		}
	}
	return fmt.Errorf("could not find this request ID in the database")
}

func (r Request) addToDividends(amount int64) error {
	if err := r.validateExists(); err != nil {
		return err
	}
	for i, request := range currentState[r.RocketID].data[r.OwnedBy].Requests {
		if request.UID == r.UID {
			currentState[r.RocketID].data[r.OwnedBy].Requests[i].DividendAmount += amount
			return nil
		}
	}
	return fmt.Errorf("could not find this request ID in the database")
}

func (r Request) validateExists() error {
	_, ok := currentState[r.RocketID]
	if !ok {
		return fmt.Errorf("could not find that rocket")
	}
	_, ok = currentState[r.RocketID].data[r.OwnedBy]
	if !ok {
		return fmt.Errorf("could not find that account in this rocket")
	}
	return nil
}
