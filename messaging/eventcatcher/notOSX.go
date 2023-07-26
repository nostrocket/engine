//go:build !darwin

package eventcatcher

func sleeper(listen chan bool) {}
