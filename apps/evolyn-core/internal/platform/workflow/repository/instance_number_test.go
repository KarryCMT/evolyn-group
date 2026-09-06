package repository

import (
	"context"
	"errors"
	"evolyn/internal/infrastructure"
	"evolyn/internal/testsupport"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestInstanceNumberFormatting(t *testing.T) {
	require.Equal(t, "WF-20260905-000001", formatInstanceNumber("20260905", 1))
	require.Equal(t, "WF-20260905-1000000", formatInstanceNumber("20260905", 1000000))
}

// 用真实 PostgreSQL 验证多连接分配不重复，且失败事务不推进计数器。
func TestInstanceNumberConcurrentAllocation(t *testing.T) {
	db := testsupport.NewPostgres(t)
	instances, _, _, _, _, _, _ := NewRuntimeRepositories(db)
	tx := infrastructure.NewTxManager(db)
	const workers = 16
	numbers := make(chan string, workers)
	failures := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			failures <- tx.WithinTransaction(context.Background(), func(ctx context.Context) error {
				number, err := instances.allocateInstanceNumber(ctx, 101)
				if err == nil {
					numbers <- number
				}
				return err
			})
		}()
	}
	wg.Wait()
	close(numbers)
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	unique := map[string]bool{}
	for number := range numbers {
		require.False(t, unique[number])
		unique[number] = true
	}
	require.Len(t, unique, workers)
	aborted := errors.New("rollback")
	var discarded, reused string
	require.ErrorIs(t, tx.WithinTransaction(context.Background(), func(ctx context.Context) error {
		var err error
		discarded, err = instances.allocateInstanceNumber(ctx, 102)
		if err != nil {
			return err
		}
		return aborted
	}), aborted)
	require.NoError(t, tx.WithinTransaction(context.Background(), func(ctx context.Context) error {
		var err error
		reused, err = instances.allocateInstanceNumber(ctx, 102)
		return err
	}))
	require.Equal(t, discarded, reused)
}
