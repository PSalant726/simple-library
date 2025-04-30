package util

import (
	"math/rand/v2"
	"strings"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RandomString returns a random alphanumeric string of length n.
func RandomString(n int) string {
	var randStr strings.Builder
	for range n {
		randStr.WriteByte(charset[rand.IntN(len(charset))])
	}

	return randStr.String()
}
