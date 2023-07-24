package state

import (
	"fmt"

	"github.com/sasha-s/go-deadlock"
)

var lock = &deadlock.Mutex{}

func Upsert(data interface{}) (any, error) {
	lock.Lock()
	defer lock.Unlock()
	switch d := data.(type) {
	case Rocket:
		return d.upsert(), nil
	}
	return nil, fmt.Errorf("no valid data type found")
}

func Flush() {
	lock.Lock()
	defer lock.Unlock()
}
