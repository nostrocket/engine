package consensustree

import (
	"github.com/nbd-wtf/go-nostr"
)

//handler
//func (s *db) upsert(key int64, val TreeEvent) {
//	if d, ok := s.data[val.StateChangeEventHeight]; !ok {
//		d = make(map[library.Sha256]TreeEvent)
//		s.data[val.StateChangeEventHeight] = d
//	}
//
//	if d, ok := s.data[val.StateChangeEventHeight][val.StateChangeEventID];
//	d, _ := s.data[val.StateChangeEventHeight]
//	d[val.StateChangeEventID] = val
//
//	for _, event := range s.data[val.Height] {
//		if event.Signer == val.Signer && event.StateChangeEventID == val.StateChangeEventID {
//			return
//		}
//	}
//	s.data[key] = append(s.data[key], val)
//}

func HandleEvent(e nostr.Event) error {
	return nil
}

//when we are replaying, we handle the consensus event BEFORE replaying the state change event, then we update the consensus data in case votepower changed. The handler should reply with the accounts that have signed this state change event hash so that the conductor can check the permille.
//the conductor should only replay the state change event after it sees the next consensus height (or end of list), this is to prevent playing state change messages until we know that there are no alternative consensus branches.

//when we are in live mode, we play the state change event, BEFORE producing the consensus event.

//fetch ALL events in the nostrocket tree, map them all.
//Once we get the EOSE, process each Kind 640064 event in order from the ignition event, find the height of each and put them into a map then handle them all, replaying state change events too.
//buffer post-EOSE incoming events in a queue which we pick up after processing everything recieved before EOSE
//don't store any state locally, replay everything from the start each time
//don't publish state until we fully sync
//

//make state changes visible in terminal logs, show which mind state was updated
