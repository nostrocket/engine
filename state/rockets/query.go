package rockets

import (
	"nostrocket/engine/library"
)

func NamesAndFounders() map[string]library.Account {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return namesAndFounders()
}

func namesAndFounders() map[string]library.Account {
	startDb()
	m := make(map[string]library.Account)
	for id, rocket := range currentState.data {
		m[id] = rocket.CreatedBy
	}
	return m
}
