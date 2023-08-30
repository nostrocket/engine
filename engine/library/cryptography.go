package library

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
		//actors.LogCLI("attempted to hash non-string or non-[]byte", 0)
	}
	h := sha256.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Random generates 32 random bytes and returns them as a hex encoded string
func Random() string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	hash := sha256.Sum256(randomBytes)
	randomString := hex.EncodeToString(hash[:])

	return randomString
}
