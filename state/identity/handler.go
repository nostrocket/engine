package identity

import (
	"encoding/json"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

func HandleEvent(event nostr.Event) (m Mapped, e error) {
	startDb()
	if sig, _ := event.CheckSignature(); !sig {
		return
	}
	if event.Kind >= 640400 && event.Kind <= 640499 {
		currentState.mutex.Lock()
		defer currentState.mutex.Unlock()
		var updates int64 = 0
		existingIdentity := getLatestIdentity(event.PubKey)
		var updateIdents []Identity
		if event.Kind == 640400 {
			var unmarshalled Kind640400
			err := json.Unmarshal([]byte(event.Content), &unmarshalled)
			if err != nil {
				return m, err
			} else {
				if len(unmarshalled.Name) > 0 {
					if existingIdentity.addName(unmarshalled.Name) {
						updates++
					}
				}
				if len(unmarshalled.About) > 0 {
					if existingIdentity.upsertBio(unmarshalled.About) {
						updates++
					}
				}
			}
		}
		if event.Kind == 640402 {
			var unmarshalled Kind640402
			err := json.Unmarshal([]byte(event.Content), &unmarshalled)
			if err != nil {
				return m, err
			}
			target, okt := currentState.data[unmarshalled.Target]
			bestower, okb := currentState.data[event.PubKey]
			var updts int64
			if okt && okb && len(target.Name) > 0 {
				if unmarshalled.Maintainer {
					if len(bestower.MaintainerBy) > 0 {
						if len(target.MaintainerBy) == 0 {
							target.MaintainerBy = event.PubKey
							updts++
						}
					}
				}
				if unmarshalled.Character {
					if _, exists := target.CharacterVouchedForBy[event.PubKey]; !exists {
						target.CharacterVouchedForBy[event.PubKey] = struct{}{}
						updts++
					}
				}
				if unmarshalled.USH {
					if len(target.UniqueSovereignBy) == 0 {
						var order int64
						for _, identity := range currentState.data {
							if identity.Order > order {
								order = identity.Order
							}
						}
						target.UniqueSovereignBy = event.PubKey
						target.Order = order + 1
						updts++
					}
				}
				if updts > 0 {
					updateIdents = append(updateIdents, target)
					updates++
				}
			}
		}
		if event.Kind == 640406 {
			var unmarshalled Kind640406
			err := json.Unmarshal([]byte(event.Content), &unmarshalled)
			if err != nil {
				return m, err
			}
			//todo validate bitcoin signed message
			existingIdentity.OpReturnAddr = append(existingIdentity.OpReturnAddr, []string{unmarshalled.Address, unmarshalled.Proof})
			updates++

		}
		if updates > 0 {
			existingIdentity.Account = event.PubKey
			currentState.data[event.PubKey] = existingIdentity
			for _, ident := range updateIdents {
				currentState.data[ident.Account] = ident
			}
			//currentState.persistToDisk()
			return getMap(), nil
		}
	}
	return m, fmt.Errorf("event %s did not cause a state change", event.ID)
}

func nameTaken(name string) bool {
	var taken bool
	for _, identity := range currentState.data {
		if identity.Name == name {
			taken = true
		}
	}
	return taken
}

func (i *Identity) addName(content string) bool {
	if len(i.Name) > 0 {
		return false
	}
	if nameTaken(content) {
		return false
	}
	if len(i.Name) > 0 {
		return false
	}
	if len(content) > 30 {
		return false
	}
	i.Name = content
	return true
}

func (i *Identity) upsertBio(content string) bool {
	if len(content) <= 560 && len(content) > 0 {
		i.About = content
		return true
	}
	return false
}
