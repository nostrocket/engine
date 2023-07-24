package state

import (
	"nostrocket/engine/library"
)

func Rockets() (m map[library.Sha256]Rocket) {
	m = make(map[library.Sha256]Rocket)
	lock.Lock()
	defer lock.Unlock()
	for sha256, rocket := range rockets {
		m[sha256] = rocket
	}
	return
}
