package subrockets

import (
	"nostrocket/engine/library"
)

func Names() map[string]library.Account {
	currentState.mutex.Lock()
	defer currentState.mutex.Unlock()
	//todo allow ignition account to create the nostrocket subrocket, rather than hardcoding it
	m := make(map[string]library.Account)
	for id, rocket := range currentState.data {
		m[id] = rocket.CreatedBy
	}
	return m
}
