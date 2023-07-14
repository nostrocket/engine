package rockets

import (
	"nostrocket/engine/library"
)

func RocketCreators() map[string]library.Account {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	return rocketCreators()
}

func rocketCreators() map[string]library.Account {
	startDb()
	m := make(map[string]library.Account)
	for rocketID, rocket := range currentState.data {
		m[rocketID] = rocket.CreatedBy
	}
	return m
}
