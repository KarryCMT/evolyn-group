package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memorySerialCounters struct {
	mu     sync.Mutex
	nextBy map[string]int64
}

func (c *memorySerialCounters) Allocate(_ context.Context, tenantID, formID uint, fieldID, cycleKey string, initial int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fmt.Sprintf("%d:%d:%s:%s", tenantID, formID, fieldID, cycleKey)
	if c.nextBy == nil {
		c.nextBy = map[string]int64{}
	}
	next, exists := c.nextBy[key]
	if !exists {
		next = initial
	}
	c.nextBy[key] = next + 1
	return next, nil
}

func (c *memorySerialCounters) Migrate() error { return nil }

func TestApplySerialNumbersUsesServerCounterAndRuleOrder(t *testing.T) {
	service := &formService{serialCounters: &memorySerialCounters{}}
	content := map[string]any{"content": map[string]any{"items": []any{
		map[string]any{"label": "客户", "widget": map[string]any{"type": "text", "widgetName": "_widget_customer", "fieldId": "abc123def4"}},
		map[string]any{"label": "订单编号", "widget": map[string]any{"type": "sn", "widgetName": "_widget_serial", "fieldId": "abc123def5", "rules": []any{
			map[string]any{"type": "literal", "value": "ORD-"},
			map[string]any{"type": "submittedAt", "format": "yyyyMMdd"},
			map[string]any{"type": "counter", "digits": 5, "fixedWidth": true, "resetCycle": "daily", "initialValue": 1},
			map[string]any{"type": "field", "fieldId": "abc123def4"},
		}}},
	}}}
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))

	first := map[string]any{"_widget_customer": "MD"}
	require.NoError(t, service.applySerialNumbers(context.Background(), 1, 2, content, first, now))
	require.Equal(t, "ORD-2026091400001MD", first["_widget_serial"])

	second := map[string]any{"_widget_customer": "KK"}
	require.NoError(t, service.applySerialNumbers(context.Background(), 1, 2, content, second, now))
	require.Equal(t, "ORD-2026091400002KK", second["_widget_serial"])
}

func TestApplySerialNumbersRejectsFixedWidthOverflow(t *testing.T) {
	service := &formService{serialCounters: &memorySerialCounters{nextBy: map[string]int64{"1:2:abc123def5:all": 1000}}}
	content := map[string]any{"content": map[string]any{"items": []any{
		map[string]any{"label": "编号", "widget": map[string]any{"type": "sn", "widgetName": "_widget_serial", "fieldId": "abc123def5", "rules": []any{
			map[string]any{"type": "counter", "digits": 3, "fixedWidth": true, "resetCycle": "none", "initialValue": 1},
		}}},
	}}}
	err := service.applySerialNumbers(context.Background(), 1, 2, content, map[string]any{}, time.Now())
	require.ErrorContains(t, err, "超过 3 位上限")
}

func TestFormatSerialDateSupportsPresetAndCustomTokens(t *testing.T) {
	now := time.Date(2026, time.September, 14, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	for format, expected := range map[string]string{
		"yyyy":       "2026",
		"yyyy/MM":    "2026/09",
		"yyyy-MM-dd": "2026-09-14",
		"MMdd":       "0914",
		"MM/dd":      "09/14",
	} {
		actual, err := formatSerialDate(now, format)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	}
	_, err := formatSerialDate(now, "YYYY-MM-DD")
	require.Error(t, err)
}
