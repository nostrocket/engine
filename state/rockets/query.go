package rockets

import (
	"nostrocket/engine/library"
	"nostrocket/state"
)

func RocketCreators() map[string]library.Account {
	startDb()
	m := make(map[string]library.Account)
	for rocketID, rocket := range state.Rockets() {
		m[rocketID] = rocket.CreatedBy
	}
	return m
}
