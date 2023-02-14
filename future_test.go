package future

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFutureThen(t *testing.T) {
	f := New(func(complete func(any), error func(error)) {
		complete(nil)
	})
	assert.NotNil(t, f)
}

func TestFutureWait(t *testing.T) {
	f1 := New(func(complete func(string), reject func(error)) {
		complete("one")
	})
	f2 := New(func(complete func(string), reject func(error)) {
		complete("two")
	})
	f3 := New(func(complete func(string), reject func(error)) {
		complete("three")
	})

	f := Wait(f1, f2, f3)

	val, err := f.Await()

	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two", "three"}, val)
	assert.NotNil(t, f)
}
