package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlattenOptionValues(t *testing.T) {
	assert.Equal(t, []string{"A"}, flattenOptionValues(" A "))
	assert.Equal(t, []string{"A", "B", "2.5"}, flattenOptionValues([]any{"A", " B ", 2.5}))
	assert.Nil(t, flattenOptionValues(nil))
}

func TestCompareOptionValueUsesNumericOrderWhenPossible(t *testing.T) {
	assert.Less(t, compareOptionValue("2", "10"), 0)
	assert.Greater(t, compareOptionValue("B", "A"), 0)
}
