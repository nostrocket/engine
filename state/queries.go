package state

import (
	"golang.org/x/exp/slices"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/state/identity"
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

func IsMaintainerOnRocket(pubkey library.Account, rocketID library.RocketID) bool {
	lock.Lock()
	defer lock.Unlock()
	return isMaintainer(pubkey, rocketID)
}

func isMaintainer(pubkey library.Account, rocketID library.RocketID) bool {
	if rocket, ok := rockets[rocketID]; ok {
		return slices.Contains(rocket.Maintainers, pubkey)
	}
	if rocketID == actors.IgnitionRocketID {
		return identity.IsMaintainer(pubkey)
	}
	return false
}
