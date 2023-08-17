package blocks

import (
	"time"

	"nostrocket/engine/library"
)

type Block struct {
	Height     int64
	Hash       library.Sha256
	MedianTime time.Time
	MinerTime  time.Time
	Difficulty int64
}

type Mapped map[int64]Block
