package blocks

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"nostrocket/engine/library"
	"nostrocket/state/merits"
)

func HandleEvent(event nostr.Event) (m Mapped, e error) {
	currentStateMu.Lock()
	defer currentStateMu.Unlock()

	if event.Kind != 1517 {
		return nil, fmt.Errorf("invalid kind")
	}

	if merits.VotepowerInNostrocketForAccount(event.PubKey) < 1 {
		return nil, fmt.Errorf("pubkey doesn't have votepower")
	}

	hash, ok := library.GetFirstTag(event, "hash")
	if !ok {
		return nil, fmt.Errorf("failed to get block hash from event")
	}

	height, ok := library.GetFirstTag(event, "height")
	if !ok {
		return nil, fmt.Errorf("failed to get block height from event")
	}

	heightInt, err := strconv.ParseInt(height, 10, 64)
	if err != nil {
		return nil, err
	}

	minerTime, ok := library.GetFirstTag(event, "minertime")
	if !ok {
		return nil, fmt.Errorf("failed to get block miner time from event")
	}
	minerTimeInt, err := strconv.ParseInt(minerTime, 10, 64)
	if err != nil {
		return nil, err
	}

	meantime, ok := library.GetFirstTag(event, "mediantime")
	if !ok {
		return nil, fmt.Errorf("failed to get block meantime from event")
	}
	meantimeInt, err := strconv.ParseInt(meantime, 10, 64)
	if err != nil {
		return nil, err
	}

	difficulty, ok := library.GetFirstTag(event, "difficulty")
	if !ok {
		return nil, fmt.Errorf("failed to get block difficulty from event")
	}
	difficultyInt, err := strconv.ParseInt(difficulty, 10, 64)
	if err != nil {
		return nil, err
	}

	if existing, exists := currentState[heightInt]; exists {
		if existing.Hash == hash {
			return nil, fmt.Errorf("we already have this block")
		}
	}
	t, ok := tip()
	if ok {
		if t.Height >= heightInt {
			return nil, fmt.Errorf("this block is not higher than our current block")
		}
	}
	currentState[heightInt] = Block{
		Height:     heightInt,
		Hash:       hash,
		MedianTime: time.Unix(meantimeInt, 0),
		MinerTime:  time.Unix(minerTimeInt, 0),
		Difficulty: difficultyInt,
	}
	return getMapped(), nil
}
