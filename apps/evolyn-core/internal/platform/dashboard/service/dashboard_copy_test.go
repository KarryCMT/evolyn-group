package service

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestDashboardCopyName(t *testing.T) {
	t.Run("appends copy suffix", func(t *testing.T) {
		assert.Equal(t, "经营总览（副本）", dashboardCopyName("经营总览"))
	})

	t.Run("preserves suffix within name limit", func(t *testing.T) {
		name := dashboardCopyName(strings.Repeat("仪", maxNameRunes))
		assert.Equal(t, maxNameRunes, utf8.RuneCountInString(name))
		assert.True(t, strings.HasSuffix(name, dashboardCopySuffix))
	})
}
