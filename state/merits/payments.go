package merits

import (
	"fmt"
	"sort"

	"nostrocket/engine/library"
)

func GetNextPaymentAddress(RocketID library.RocketID, amount int64) (a library.Account, e error) {
	startDb()
	currentStateMu.Lock()
	defer currentStateMu.Unlock()
	return getNextPaymentAddress(RocketID, amount)
}

func getNextPaymentAddress(RocketID library.RocketID, amount int64) (a library.Account, e error) {
	merits, exist := currentState[RocketID]
	if !exist {
		return a, fmt.Errorf("could not find rocket with that ID")
	}
	account, ok := getNextRemunerationAddress(merits, amount)
	if ok {
		return account, nil
	}
	account, ok = getNextDividendAddress(merits, amount)
	if ok {
		return account, nil
	}
	return a, fmt.Errorf("could not find the next payment address")
}

func getNextDividendAddress(merits meritsForRocket, amount int64) (a library.Account, unpaid bool) {
	for _, request := range merits.orderedDividendRequests() {
		if amount < request.Amount {
			a = request.OwnedBy
			unpaid = true
		}
	}
	return
}

func getNextRemunerationAddress(merits meritsForRocket, amount int64) (a library.Account, unpaid bool) {
	for _, request := range merits.orderedRemunerationRequests() {
		if amount < request.Amount {
			a = request.OwnedBy
			unpaid = true
		}
	}
	return
}

func (mfr meritsForRocket) orderedRemunerationRequests() (r []Request) {
	for _, merit := range mfr.data {
		for _, request := range merit.Requests {
			if request.Approved {
				if request.Amount > request.RemuneratedAmount {
					r = append(r, request)
				}
			}
		}
	}
	sort.Slice(r, func(i, j int) bool {
		return r[i].Nth < r[j].Nth
	})
	return
}

func (mfr meritsForRocket) orderedDividendRequests() (r []Request) {
	for _, merit := range mfr.data {
		for _, request := range merit.Requests {
			if request.Approved {
				if request.DividendAmount == 0 {
					request.DividendAmount = 1
				}
				r = append(r, request)
			}
		}
	}
	sort.Slice(r, func(i, j int) bool {
		return (r[i].Amount / r[i].DividendAmount) > (r[j].Amount / r[j].DividendAmount)
	})
	return
}
