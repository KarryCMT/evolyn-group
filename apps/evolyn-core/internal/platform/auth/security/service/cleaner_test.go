package service

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestSessionCleanupWorkerDefaults 防止 server 装配时遗漏默认扫描周期；Worker
// 的实际数据库删除逻辑由 repository 集成测试覆盖。
func TestSessionCleanupWorkerDefaults(t *testing.T) {
	worker := NewSessionCleanupWorker(&fakeSessionRepo{}, 0, logrus.New())
	assert.Equal(t, defaultSessionCleanupInterval, worker.interval)
}

// TestNilSessionCleanupWorkerRun 防御性保障：即使未来装配回归为 nil，后台任务
// 也应静默退出而不是让整个 API 进程 panic。
func TestNilSessionCleanupWorkerRun(t *testing.T) {
	var worker *SessionCleanupWorker
	done := make(chan struct{})
	go func() {
		worker.Run(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("nil cleanup worker did not return")
	}
}
