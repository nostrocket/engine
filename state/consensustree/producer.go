package consensustree

//var lock = &deadlock.Mutex{}

//func ProduceEvent(stateChangeEventID library.Sha256, bitcoinHeight int64) (nostr.Event, error) {
//	//todo: problem: we will calculate votepower incorrectly on events that increase votepower becuase we increase votepower and THEN produce a consensus event.
//	//simplest solution: store the last votepower increase and consensus height and eventID of the event that did it in memory when handling an event that increases votewpoer. If we are at the same height
//	//and producing a consensus event for the event that just increased votepower, deduct the amount of votepower stored in memory for the account that increased it, and use that as the current votepower
//	lock.Lock()
//	defer lock.Unlock()
//	currentState.mutex.Lock()
//	defer currentState.mutex.Unlock()
//	//todo problem: newly created votepower doesn't produce consensenustree events
//	return produceEvent(stateChangeEventID, bitcoinHeight)
//}

//func produceEvent(stateChangeEventID library.Sha256, bitcoinHeight int64) (nostr.Event, error) {
//	var t = nostr.Tags{}
//	var eventHeight int64
//	t = append(t, nostr.Tag{"e", actors.IgnitionEvent, "", "root"})
//	if len(currentState.data) == 0 && actors.MyWallet().Account == actors.IgnitionAccount {
//		eventHeight = 1
//		t = append(t, nostr.Tag{"e", actors.ConsensusTree, "", "reply"})
//	} else {
//		var heighest int64
//		var eID library.Sha256
//		//find the latest stateChangeEvent that we have signed
//		for i, m := range currentState.data {
//			//todo: what about if we are new votepower and haven't signed anything yet?
//			for sha256, event := range m {
//				if event.IHaveSigned {
//					if i >= heighest && !event.IHaveReplaced {
//						eID = sha256
//						heighest = i
//					}
//				}
//			}
//		}
//		eventHeight = heighest
//		eventHeight++
//		if len(eID) != 64 {
//			return nostr.Event{}, fmt.Errorf("could not find latest state change event")
//		}
//		t = append(t, nostr.Tag{"e", eID, "", "reply"})
//	}
//
//	j, err := json.Marshal(Kind640001{
//		StateChangeEventID: stateChangeEventID,
//		Height:             eventHeight,
//		BitcoinHeight:      bitcoinHeight,
//	})
//	if err != nil {
//		return nostr.Event{}, err
//	}
//	n := nostr.Event{
//		PubKey:    actors.MyWallet().Account,
//		CreatedAt: nostr.Timestamp(time.Now().Unix()),
//		Kind:      640001,
//		Tags:      t,
//		Content:   fmt.Sprintf("%s", j),
//	}
//	n.ID = n.GetID()
//	n.Sign(actors.MyWallet().PrivateKey)
//	return n, nil
//}
