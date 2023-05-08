package mirv

import (
	"nostrocket/engine/library"
)

func Names() map[string]library.Account {
	startDb()
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	//todo allow ignition account to create the nostrocket mirv, rather than hardcoding it
	m := make(map[string]library.Account)
	for id, rocket := range currentState.data {
		m[id] = rocket.CreatedBy
	}
	return m
}
