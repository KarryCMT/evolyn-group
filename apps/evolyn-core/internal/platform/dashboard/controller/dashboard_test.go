package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"

	dashboarderrors "evolyn/internal/platform/dashboard"
	"evolyn/internal/platform/httpx"

	"github.com/stretchr/testify/assert"
)

func TestDashboardQueryResultClass(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "success", want: "success"},
		{name: "timeout through biz unwrap", err: httpx.Wrap(dashboarderrors.ErrQueryLimitExceeded, context.DeadlineExceeded), want: "timeout"},
		{name: "limit", err: dashboarderrors.ErrQueryLimitExceeded, want: "limit_exceeded"},
		{name: "invalid", err: fmt.Errorf("wrapped: %w", dashboarderrors.ErrQueryInvalid), want: "invalid"},
		{name: "forbidden", err: dashboarderrors.ErrForbidden, want: "forbidden"},
		{name: "other biz", err: dashboarderrors.ErrDraftConflict, want: "biz_409"},
		{name: "internal", err: errors.New("database unavailable"), want: "internal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, dashboardQueryResultClass(test.err))
		})
	}
}
