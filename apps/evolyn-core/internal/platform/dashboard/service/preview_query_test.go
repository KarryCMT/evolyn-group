package service

import (
	"context"
	"errors"
	"testing"
	"time"

	queryengine "evolyn/internal/engine/query"
	dashboarderrors "evolyn/internal/platform/dashboard"
	"evolyn/internal/platform/dashboard/model"
	iammodel "evolyn/internal/platform/iam/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingDashboardQueryExecutor struct{}

type staticDashboardAccess map[string]bool

func (a staticDashboardAccess) Permissions(context.Context, *iammodel.User) map[string]bool {
	return a
}

func (blockingDashboardQueryExecutor) Execute(ctx context.Context, _ *iammodel.User, _ string, _ queryengine.LogicalPlan) (*DashboardQueryResultView, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestExecutePreviewPlanTimeoutAndConcurrencyLimit(t *testing.T) {
	svc := &dashboardService{
		queries:      blockingDashboardQueryExecutor{},
		queryTimeout: 5 * time.Millisecond,
		querySlots:   make(chan struct{}, 1),
	}

	_, err := svc.executePreviewPlan(context.Background(), &iammodel.User{}, "form_demo", queryengine.LogicalPlan{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, dashboarderrors.ErrQueryLimitExceeded))
	assert.True(t, errors.Is(err, context.DeadlineExceeded))

	svc.querySlots <- struct{}{}
	_, err = svc.executePreviewPlan(context.Background(), &iammodel.User{}, "form_demo", queryengine.LogicalPlan{})
	<-svc.querySlots
	require.Error(t, err)
	assert.True(t, errors.Is(err, dashboarderrors.ErrQueryLimitExceeded))
	assert.NotErrorIs(t, err, context.DeadlineExceeded)
}

func TestRuntimeWidgetQueryUsesViewPermission(t *testing.T) {
	member := &iammodel.User{ID: 1}
	request := &model.RuntimeQueryRequest{}

	denied := &dashboardService{access: staticDashboardAccess{}}
	_, err := denied.RuntimeWidgetQuery(context.Background(), member, "dashboard_demo", "widget_demo", request)
	require.Error(t, err)
	assert.ErrorIs(t, err, dashboarderrors.ErrForbidden)

	viewOnly := &dashboardService{access: staticDashboardAccess{"dashboard-actions:view": true}}
	_, err = viewOnly.RuntimeWidgetQuery(context.Background(), member, "dashboard_demo", "widget_demo", request)
	require.Error(t, err)
	assert.ErrorIs(t, err, dashboarderrors.ErrQueryInvalid)
	assert.NotErrorIs(t, err, dashboarderrors.ErrForbidden)
}
