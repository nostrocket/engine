package library

import (
	"crypto/sha256"
	"fmt"
)

func Sha256Sum(data interface{}) Sha256 {
	var b []byte
	switch d := data.(type) {
	case string:
		b = []byte(d)
	case []byte:
		b = d
	default:
		LogCLI("attempted to hash non-string or non-[]byte", 0)
	}
	h := sha256.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}
