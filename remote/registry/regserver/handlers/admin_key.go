package handlers

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewAdminKey generates a randomized key for admin work
// this is a lazy stopgap for now
func NewAdminKey() string {
	return randStringRunes(25)
}

// randStringRunes creates a random string of n length
func randStringRunes(n int) string {
	runes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}
