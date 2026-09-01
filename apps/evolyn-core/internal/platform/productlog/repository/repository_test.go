package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLikePatternEscapesWildcards 关键词按字面匹配：%/_/\\ 不作通配符
func TestLikePatternEscapesWildcards(t *testing.T) {
	assert.Equal(t, `%\%100%`, likePattern("%100"))
	assert.Equal(t, `%a\_b%`, likePattern("a_b"))
	assert.Equal(t, `%a\\b%`, likePattern(`a\b`))
	assert.Equal(t, `%采购申请%`, likePattern("采购申请"))
}
