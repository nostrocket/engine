package subrockets

import (
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func Names() map[string]library.Account {
	//todo allow ignition account to create the nostrocket subrocket, rather than hardcoding it
	m := make(map[string]library.Account)
	m["nostrocket"] = actors.IgnitionAccount
	return m
}
