//go:build !darwin

package relays

func sleeper(listen chan bool) {}
