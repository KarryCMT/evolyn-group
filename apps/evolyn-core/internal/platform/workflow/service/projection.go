// 流程状态投影（物理表存储方案 §10.2）：流程实例状态变更时在同一事务内
// 经窄端口刷新表单记录信封与物理表的查询投影。铁律：wf_instance.status 是
// 唯一事实源，投影只读复制、绝不反向驱动流程；流程内核不感知本端口，
// 平台 Service/Worker 在引擎推进成功后调用。
package service

import (
	"context"
	"strconv"
	"time"

	wfapp "evolyn/internal/platform/workflow"
	wfmodel "evolyn/internal/platform/workflow/model"
	"evolyn/internal/platform/workflow/repository"

	"gorm.io/gorm"
)

// FormProjectionPort 表单投影窄端口（由 Form Domain 的
// WorkflowProjectionUpdater 经装配层适配）：status 为实例状态枚举字符串。
type FormProjectionPort interface {
	UpdateWorkflowProjection(ctx context.Context, recordID uint, instanceNo, status string, updatedAt time.Time) error
}

// FormProjector 投影执行器：按实例绑定（business_type=form_record）解析
// 记录并委托端口；非表单业务或端口未注入时幂等空跑。
type FormProjector struct {
	port   FormProjectionPort
	reader repository.RuntimeReader
}

// NewFormProjector 构造投影执行器（port/reader 为 nil 时空跑，便于单测桩）。
func NewFormProjector(port FormProjectionPort, reader repository.RuntimeReader) *FormProjector {
	return &FormProjector{port: port, reader: reader}
}

// Project 在实例状态变更的当前事务内刷新表单投影；实例不存在返回原始
// 错误（视为异常，调用方事务回滚），非表单业务空跑。
func (p *FormProjector) Project(ctx context.Context, instanceID uint) error {
	if p == nil || p.port == nil || instanceID == 0 {
		return nil
	}
	instance, err := p.reader.FindInstanceRow(ctx, instanceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound { //nolint:errorlint // reader 返回裸 sentinel
			return nil // 实例已不存在（清理竞态）：投影无从谈起，不阻断调用方事务
		}
		return err
	}
	if instance.BusinessType != "form_record" {
		return nil
	}
	recordID, err := strconv.ParseUint(instance.BusinessID, 10, 64)
	if err != nil || recordID == 0 {
		//nolint:nilerr // 历史绑定的非数字业务键：无记录可投影，跳过
		return nil
	}
	if err := p.port.UpdateWorkflowProjection(ctx, uint(recordID), instance.InstanceNo, instance.Status, time.Now()); err != nil {
		return wfapp.WrapProjectionFailed(err)
	}
	return nil
}

// RecalibrateByForm 管理员受控的投影校准（方案 §10.2）：以 wf_instance 为源
// 按表单分批扫描实例并覆盖写回记录投影（单号/状态/时间），用于上线回填、
// 故障修复与一致性巡检。只修复投影，绝不反向修改流程实例；返回校准的
// 实例数。分批自治推进（每批独立事务），任一批失败即返回。
func (p *FormProjector) RecalibrateByForm(ctx context.Context, tx TxManager, formID uint, batchSize int) (int, error) {
	if p == nil || p.port == nil {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	total := 0
	afterID := uint(0)
	for {
		batch, hasMore, err := p.reader.ListInstanceRowsByForm(ctx, formID, batchSize, afterID)
		if err != nil {
			return total, err
		}
		if len(batch) == 0 {
			return total, nil
		}
		for i := range batch {
			instance := batch[i]
			if err := tx.WithinTransaction(ctx, func(tctx context.Context) error {
				return p.projectRow(tctx, &instance)
			}); err != nil {
				return total, err
			}
			total++
		}
		if !hasMore {
			return total, nil
		}
		afterID = batch[len(batch)-1].ID
	}
}

// projectRow 直接按实例行投影（校准路径绕过再查实例行）。
func (p *FormProjector) projectRow(ctx context.Context, instance *wfmodel.WfInstance) error {
	if instance.BusinessType != "form_record" {
		return nil
	}
	recordID, err := strconv.ParseUint(instance.BusinessID, 10, 64)
	if err != nil || recordID == 0 {
		//nolint:nilerr // 历史绑定的非数字业务键：无记录可投影，跳过
		return nil
	}
	if err := p.port.UpdateWorkflowProjection(ctx, uint(recordID), instance.InstanceNo, instance.Status, time.Now()); err != nil {
		return wfapp.WrapProjectionFailed(err)
	}
	return nil
}
