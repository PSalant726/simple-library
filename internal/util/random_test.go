package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomString(t *testing.T) {
	testStr := RandomString(30)

	t.Run("returns a string", func(t *testing.T) {
		t.Run("with the expected length", func(t *testing.T) {
			assert.Len(t, testStr, 30)
		})

		t.Run("with only the expected characters", func(t *testing.T) {
			assert.Subset(t, []rune(charset), []rune(testStr))
		})
	})
}
