package identity

import (
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
)

func HandleEvent(event nostr.Event) (m Mapped, e error) {
	startDb()
	if sig, _ := event.CheckSignature(); !sig {
		return
	}
	if event.Kind == 1 {
		return handleByTags(event)
	}
	return m, fmt.Errorf("event %s did not cause a state change", event.ID)
}

func handleByTags(event nostr.Event) (m Mapped, e error) {
	if operation, ok := library.GetFirstTag(event, "op"); ok {
		ops := strings.Split(operation, ".")
		if len(ops) > 2 {
			if ops[1] == "identity" {
				currentState.mutex.Lock()
				defer currentState.mutex.Unlock()
				switch o := ops[2]; {
				case o == "permanym":
					return handlePermanym(event)
				case o == "ush":
					return handleUsh(event)
				}
			}
		}
	}
	return nil, fmt.Errorf("no valid operation found 543c2345")
}

func handleUsh(event nostr.Event) (m Mapped, e error) {
	proposerIdentity := getLatestIdentity(event.PubKey)
	if len(proposerIdentity.UniqueSovereignBy) == 0 {
		return nil, fmt.Errorf("proposer is not in the identity tree")
	}
	targetIdentity, ok := library.GetOpData(event)
	if !ok {
		return nil, fmt.Errorf("event does not contain the pubkey of a target account")
	}
	if len(targetIdentity) != 64 {
		return nil, fmt.Errorf("invalid pubkey for target account")
	}
	existingIdentity := getLatestIdentity(targetIdentity)
	if len(existingIdentity.Name) == 0 {
		return nil, fmt.Errorf("target account does not have a permanym")
	}
	if len(existingIdentity.UniqueSovereignBy) > 0 {
		return nil, fmt.Errorf("target account is already in the identity tree")
	}
	var order int64
	for _, identity := range currentState.data {
		if identity.Order > order {
			order = identity.Order
		}
	}
	existingIdentity.UniqueSovereignBy = event.PubKey
	existingIdentity.Order = order + 1
	err := existingIdentity.upsert(existingIdentity.Account)
	if err != nil {
		return nil, err
	}
	return getMap(), nil
}

func handlePermanym(event nostr.Event) (m Mapped, e error) {
	existingIdentity := getLatestIdentity(event.PubKey)
	if len(existingIdentity.Name) == 0 {
		if permanym, ok := library.GetOpData(event); ok {
			if len(permanym) < 21 {
				err := existingIdentity.addName(permanym)
				if err != nil {
					return nil, err
				}
				existingIdentity.PermanymEventID = event.ID
				err = existingIdentity.upsert(event.PubKey)
				if err != nil {
					return nil, err
				}
				return getMap(), nil
			}
			return nil, fmt.Errorf("permanym length is too long")
		}
		return nil, fmt.Errorf("event did not contain a permanym")
	}
	return nil, fmt.Errorf("account already has a permanym")
}

//
//
//{
//	if event.Kind >= 640400 && event.Kind <= 640499 {
//
//		var updates int64 = 0
//		existingIdentity := getLatestIdentity(event.PubKey)
//
//		if event.Kind == 640402 {
//			var unmarshalled Kind640402
//			err := json.Unmarshal([]byte(event.Content), &unmarshalled)
//			if err != nil {
//				return m, err
//			}
//			target, okt := currentState.data[unmarshalled.Target]
//			bestower, okb := currentState.data[event.PubKey]
//			var updts int64
//			if okt && okb && len(target.Name) > 0 {
//				if unmarshalled.Maintainer {
//					if len(bestower.MaintainerBy) > 0 {
//						if len(target.MaintainerBy) == 0 {
//							target.MaintainerBy = event.PubKey
//							updts++
//						}
//					}
//				}
//				if unmarshalled.Character {
//					if _, exists := target.CharacterVouchedForBy[event.PubKey]; !exists {
//						target.CharacterVouchedForBy[event.PubKey] = struct{}{}
//						updts++
//					}
//				}
//				if unmarshalled.USH {
//					if len(target.UniqueSovereignBy) == 0 {
//						var order int64
//						for _, identity := range currentState.data {
//							if identity.Order > order {
//								order = identity.Order
//							}
//						}
//						target.UniqueSovereignBy = event.PubKey
//						target.Order = order + 1
//						updts++
//					}
//				}
//				if updts > 0 {
//					updateIdents = append(updateIdents, target)
//					updates++
//				}
//			}
//		}
//		if event.Kind == 640406 {
//			var unmarshalled Kind640406
//			err := json.Unmarshal([]byte(event.Content), &unmarshalled)
//			if err != nil {
//				return m, err
//			}
//			//todo validate bitcoin signed message
//			existingIdentity.OpReturnAddr = append(existingIdentity.OpReturnAddr, []string{unmarshalled.Address, unmarshalled.Proof})
//			updates++
//
//		}
//		if updates > 0 {
//			existingIdentity.Account = event.PubKey
//			currentState.data[event.PubKey] = existingIdentity
//			for _, ident := range updateIdents {
//				currentState.data[ident.Account] = ident
//			}
//			//currentState.persistToDisk()
//			return getMap(), nil
//		}
//	}
//
//}

func (i *Identity) upsert(pubkey string) error {
	if len(i.Account) == 64 {
		if i.Account != pubkey {
			return fmt.Errorf("wrong pubkey for account")
		}
	}
	i.Account = pubkey
	currentState.data[i.Account] = *i
	return nil
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

func (i *Identity) addName(content string) error {
	if len(i.Name) > 0 {
		return fmt.Errorf("account has a permanym")
	}
	if nameTaken(content) {
		return fmt.Errorf("permanym is already taken")
	}
	if len(content) > 30 {
		return fmt.Errorf("permanym is too long")
	}
	i.Name = content
	return nil
}
