// Package service 表单资产域服务（后端契约 §2）：表单 CRUD、草稿保存（乐观锁）、
// 不可变发布、运行时 bootstrap 与记录提交。所有写路径在 Service 内按权限集复核
// （与鉴权中间件同口径），值校验以发布快照为准（服务端最终裁决）。
package service

import (
	"context"

	"evolyn/internal/platform/form/model"
	iammodel "evolyn/internal/platform/iam/model"
)

// AccessEvaluator 权限集窄端口（装配层由 application 域 RBAC 评估器适配）。
type AccessEvaluator interface {
	Permissions(ctx context.Context, member *iammodel.User) map[string]bool
}

// MenuMaintenance 表单资产菜单节点维护窄端口（M2-资产-1）：由 application
// 域在装配层适配；表单域在创建/改名/删除的事务内调用，菜单节点写入与
// menu_revision 递增随之加入同一事务（跨域经窄端口，域间不直接依赖）。
type MenuMaintenance interface {
	// AttachFormEntry 表单创建事务内挂 form 资产节点；parentEntryCode 为空
	// 挂应用根级，非空挂同应用指定分组下（非法分组返回 APP_MENU_PARENT_INVALID）
	AttachFormEntry(ctx context.Context, applicationID, formID uint, name, parentEntryCode string) error
	// SyncFormEntryName 表单改名事务内同步节点展示名
	SyncFormEntryName(ctx context.Context, applicationID, formID uint, name string) error
	// SyncFormEntryAppearance 表单图标/颜色修改事务内同步节点展示属性
	//（ADR-011：展示属性以资产域为事实源；空串清空，出网投影为 null）
	SyncFormEntryAppearance(ctx context.Context, applicationID, formID uint, icon, color string) error
	// DetachFormEntry 表单删除事务内软删节点
	DetachFormEntry(ctx context.Context, applicationID, formID uint) error
}

// ApplicationDirectory 应用域只读窄端口（装配层由 application 仓储适配）：
// 表单归属校验与运行时 bootstrap 使用，form 域不直接依赖 application 域。
type ApplicationDirectory interface {
	// ApplicationByID 按应用 ID 取只读视图（ctx 租户过滤：跨租户即 notFound）
	ApplicationByID(ctx context.Context, id uint) (app ApplicationView, notFound bool, err error)
	// ApplicationByCode 按应用编码取只读视图
	ApplicationByCode(ctx context.Context, code string) (app ApplicationView, notFound bool, err error)
}

// ApplicationView 应用只读视图（form 域关心的最小字段）。
type ApplicationView struct {
	ID     uint
	Status string
}

// FormReference 引用视图条目（ADR-011「查看引用视图」）：表单被哪个应用
// 的哪个菜单节点引用。字段为出网 DTO 直接复用（form 域自有词汇，不依赖
// application 域模型）。
type FormReference struct {
	ApplicationCode string  `json:"applicationCode"`
	ApplicationName string  `json:"applicationName"`
	EntryID         string  `json:"entryId"`
	EntryName       string  `json:"entryName"`
	ParentEntryID   *string `json:"parentEntryId"`
}

// ReferenceSource 引用视图只读窄端口（装配层由 application 域菜单仓储
// 适配）：跨应用反查引用指定表单的菜单节点；端口未注入（单测）时引用
// 视图返回空集。
type ReferenceSource interface {
	// ListFormReferences 跨应用反查引用指定表单的菜单节点（ctx 租户过滤）
	ListFormReferences(ctx context.Context, formID uint) ([]FormReference, error)
}

// FormService 表单资产域服务接口。
type FormService interface {
	// Create 创建表单资产（事务内配额占位；草稿初始化为空协议文档）
	Create(ctx context.Context, member *iammodel.User, req *model.CreateFormRequest) (*model.FormDetail, error)
	// List 应用内表单游标分页
	List(ctx context.Context, member *iammodel.User, query model.ListFormsQuery) (*model.FormPage, error)
	// Get 按公开编码读取表单详情（含草稿全文与修订口令）
	Get(ctx context.Context, member *iammodel.User, code string) (*model.FormDetail, error)
	// Update 白名单更新名称/图标/颜色（名称与展示属性以本域为事实源）
	Update(ctx context.Context, member *iammodel.User, code string, req *model.UpdateFormRequest) (*model.FormDetail, error)
	// SaveDraft 保存草稿：严格协议校验 + 乐观锁条件更新
	SaveDraft(ctx context.Context, member *iammodel.User, code string, req *model.SaveDraftRequest) (*model.SaveDraftResult, error)
	// Delete 软删表单（发布版本保留）
	Delete(ctx context.Context, member *iammodel.User, code string) error
	// Publish 发布：白名单 + 严格校验 → 事务内创建不可变快照并回写 latest
	Publish(ctx context.Context, member *iammodel.User, code string, req *model.PublishRequest) (*model.PublishResult, error)
	// SwitchType 切换表单类型（ADR-011，form-actions:switch-type 动作复核）：
	// 流程表单切标准后原流程数据保留，仅不可再发起流程
	SwitchType(ctx context.Context, member *iammodel.User, code string, req *model.SwitchFormTypeRequest) (*model.FormDetail, error)
	// Copy 复制表单（ADR-011）：同应用 copy-in-app / 跨应用 copy-cross-app
	// 双动作码，复制草稿全文并挂目标应用菜单，事务内占目标配额
	Copy(ctx context.Context, member *iammodel.User, code string, req *model.CopyFormRequest) (*model.FormDetail, error)
	// ListReferences 查看引用视图（ADR-011）：表单被哪些应用菜单引用
	ListReferences(ctx context.Context, member *iammodel.User, code string) ([]FormReference, error)
	// GetRuntime 运行时 bootstrap（appCode 归属 + 已发布校验；普通成员可读）：
	// 权限组判定入口（view ∨ add），出网追加 permissions 投影
	GetRuntime(ctx context.Context, member *iammodel.User, appCode, formCode string) (*model.FormRuntime, error)
	// SubmitRecord 提交记录：按 (publishedVersion, schemaRevision) 定位快照并按其终审
	SubmitRecord(ctx context.Context, member *iammodel.User, req *model.SubmitRecordRequest) (*model.SubmitRecordResult, error)
}

// PermissionEvaluatorInjector 装配期注入能力（可选）：权限组判定器（表单权限
// P1）。未注入时执行点按 S4 基线放行（存量行为零变更），便于单测桩与灰度装配。
type PermissionEvaluatorInjector interface {
	UsePermissionEvaluator(evaluator FormPermissionEvaluator)
}
