package library

import (
	"github.com/sasha-s/go-deadlock"
)

func ValidateSaneExecutionTime() func() {
	mu := deadlock.Mutex{}
	mu.Lock()
	go func() {
		mu.Lock()
		mu.Unlock()
	}()
	return func() {
		mu.Unlock()
	}
}
