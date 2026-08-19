package repository

import (
	"context"
	"strconv"
	"time"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/tenant/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tenantRepository struct {
	db  *gorm.DB
	rdb *infrastructure.RedisDB
}

// LifecycleTimes 注销生命周期时间线（FIX-012）：deleted 状态由服务层计算写入
type LifecycleTimes struct {
	DeleteRequestedAt time.Time
	RetentionUntil    time.Time
}

// TenantRepository 租户域数据访问；管理面 CRUD 随 P3 运营域接口补充
type TenantRepository interface {
	GetByID(ctx context.Context, id uint) (*model.Tenant, error)
	GetByIDs(ctx context.Context, ids []uint) ([]model.Tenant, error)
	GetByCode(ctx context.Context, code string) (*model.Tenant, error)
	// GetStatus 租户当前状态（FIX-007 请求级拦截用）：短缓存 + 变更失效，
	// 注销行不物理删除（墓碑 + retention），读取不过滤软删行
	GetStatus(ctx context.Context, id uint) (string, error)
	SeedDefaultTenant() error
	Migrate() error

	// 运营面 CRUD（P3-1）：内部剥离租户上下文，仅限 /platform 域调用
	List(ctx context.Context) ([]model.Tenant, error)
	Create(ctx context.Context, tenant *model.Tenant) (*model.Tenant, error)
	Update(ctx context.Context, tenant *model.Tenant) (*model.Tenant, error)
	// UpdateStatus 生命周期流转（FIX-012）：deleted 不再软删整行（保留墓碑
	// 供审计/恢复），active 恢复时清空注销时间线；同步失效状态缓存
	UpdateStatus(ctx context.Context, id uint, status string, lifecycle *LifecycleTimes) error

	// ListPurgeable 到达保留截止且未清理的注销租户（FIX-012，Purge Worker 用）
	ListPurgeable(ctx context.Context, now time.Time) ([]model.Tenant, error)
	// PurgeTenantData 物理清理租户业务数据并落墓碑标记（FIX-012）：
	// 成员/分组/角色/部门及其关系行硬删，租户行保留 code 防重用
	PurgeTenantData(ctx context.Context, tenantID uint) error
}

// NewRepository 租户域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB, rdb *infrastructure.RedisDB) TenantRepository {
	return &tenantRepository{
		db:  db,
		rdb: rdb,
	}
}

// 租户状态缓存（FIX-007）：Hash 结构，field=租户 ID，value=status。
// 写路径（状态/配置变更）显式失效；Redis 禁用时每次回源查库
const tenantStatusCacheKey = "tenants:status"

// withContext 以请求 ctx 打开新会话。FIX-014 后 tenants 表不再带 tenant_id
// 列，GORM 租户 Callback 对本表查询不生效，运营面/登录前查询天然无自我过滤
func (t *tenantRepository) withContext(ctx context.Context) *gorm.DB {
	return t.db.WithContext(ctx)
}

func (t *tenantRepository) GetByID(ctx context.Context, id uint) (*model.Tenant, error) {
	tenant := new(model.Tenant)
	if err := t.withContext(ctx).First(tenant, id).Error; err != nil {
		return nil, err
	}
	return tenant, nil
}

// GetByIDs 批量取租户（成员关系列表组装用）；ids 为空返回空集
func (t *tenantRepository) GetByIDs(ctx context.Context, ids []uint) ([]model.Tenant, error) {
	tenants := make([]model.Tenant, 0)
	if len(ids) == 0 {
		return tenants, nil
	}
	if err := t.withContext(ctx).Where("id IN ?", ids).Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

func (t *tenantRepository) GetByCode(ctx context.Context, code string) (*model.Tenant, error) {
	tenant := new(model.Tenant)
	if err := t.withContext(ctx).Where("code = ?", code).First(tenant).Error; err != nil {
		return nil, err
	}
	return tenant, nil
}

// GetStatus 请求链租户状态检查（FIX-007）：优先读缓存；未命中回源时
// Unscoped 读取——注销租户行是墓碑而非软删行，必须可见才能返回 deleted
func (t *tenantRepository) GetStatus(ctx context.Context, id uint) (string, error) {
	field := strconv.FormatUint(uint64(id), 10)
	status := ""
	if err := t.rdb.HGet(tenantStatusCacheKey, field, &status); err == nil && status != "" {
		return status, nil
	}

	// status 列 NOT NULL，Pluck 不会产生 NULL 元素；空结果即租户不存在
	statuses := make([]string, 0, 1)
	if err := t.withContext(ctx).Unscoped().
		Model(&model.Tenant{}).Where("id = ?", id).Pluck("status", &statuses).Error; err != nil {
		return "", err
	}
	if len(statuses) == 0 {
		return "", gorm.ErrRecordNotFound
	}
	status = statuses[0]

	// 回源后回填缓存（失败不影响主流程）
	_ = t.rdb.HSet(tenantStatusCacheKey, field, status)
	return status, nil
}

func (t *tenantRepository) Migrate() error {
	return t.db.AutoMigrate(&model.Tenant{})
}

// SeedDefaultTenant 确保默认租户存在，并将所有租户列为 0/NULL 的存量行归属到默认租户。
// 必须在其余模型 AutoMigrate（tenant_id 列已带 default:1 回填）之后调用：
// 若默认租户实际 ID 不是 1（如表经历过清理），此处负责把 1 修正为真实 ID。
func (t *tenantRepository) SeedDefaultTenant() error {
	tenant := &model.Tenant{
		Code:   model.DefaultTenantCode,
		Name:   "默认租户",
		Plan:   "free",
		Status: model.TenantActive,
	}
	if err := t.db.Clauses(clause.OnConflict{DoNothing: true}).Create(tenant).Error; err != nil {
		return err
	}

	created := new(model.Tenant)
	if err := t.db.Where("code = ?", model.DefaultTenantCode).First(created).Error; err != nil {
		return err
	}

	if created.ID == model.DefaultTenantID {
		return nil
	}

	// 默认租户 ID 与列默认值不一致时，统一把存量数据修正到真实 ID（PostgreSQL 不支持多表 UPDATE，逐表执行）
	for _, table := range []string{"users", "groups", "roles", "departments"} {
		if err := t.db.Exec(
			"UPDATE "+table+" SET tenant_id = ? WHERE tenant_id = ? OR tenant_id IS NULL",
			created.ID, model.DefaultTenantID,
		).Error; err != nil {
			return err
		}
	}

	return nil
}

// platformCtx 运营面专用：刻意剥离请求携带的租户上下文（历史版本 tenants 表
// 曾带 tenant_id 列，运营者自身会话的租户上下文会造成自我过滤；FIX-014 后
// 列已移除，保留剥离以明确「平台域不依赖租户上下文」的语义），并施加独立超时
func platformCtx(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// invalidateStatusCache 状态缓存失效（状态/配置写路径调用）
func (t *tenantRepository) invalidateStatusCache(id uint) {
	_ = t.rdb.HDel(tenantStatusCacheKey, strconv.FormatUint(uint64(id), 10))
}

// List 运营面租户列表：注销租户默认不返回（运营如需查看墓碑走 Get）
func (t *tenantRepository) List(ctx context.Context) ([]model.Tenant, error) {
	tenants := make([]model.Tenant, 0)
	pctx, cancel := platformCtx(ctx)
	defer cancel()
	if err := t.db.WithContext(pctx).
		Where("status <> ?", model.TenantDeleted).
		Order("id").Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

// Create 开通租户（运营面）：OwnerAccountId 由服务层写入（FIX-016 可空 FK）
func (t *tenantRepository) Create(ctx context.Context, tenant *model.Tenant) (*model.Tenant, error) {
	pctx, cancel := platformCtx(ctx)
	defer cancel()
	if err := t.db.WithContext(pctx).Create(tenant).Error; err != nil {
		return nil, err
	}
	t.invalidateStatusCache(tenant.ID)
	return tenant, nil
}

// Update 运营面更新（名称/套餐/配置/配额覆盖/归属账号）
func (t *tenantRepository) Update(ctx context.Context, tenant *model.Tenant) (*model.Tenant, error) {
	pctx, cancel := platformCtx(ctx)
	defer cancel()
	if err := t.db.WithContext(pctx).Model(&model.Tenant{}).Where("id = ?", tenant.ID).
		Select("name", "plan", "owner_account_id", "config", "quotas").Updates(tenant).Error; err != nil {
		return nil, err
	}
	t.invalidateStatusCache(tenant.ID)
	return tenant, nil
}

// UpdateStatus 生命周期流转（FIX-012）：不再对 deleted 做软删（保留墓碑行，
// 保留期内运营可查库恢复）；deleted 写入注销时间线，active 恢复清空时间线
func (t *tenantRepository) UpdateStatus(ctx context.Context, id uint, status string, lifecycle *LifecycleTimes) error {
	pctx, cancel := platformCtx(ctx)
	defer cancel()
	db := t.db.WithContext(pctx)

	updates := map[string]interface{}{"status": status}
	switch status {
	case model.TenantDeleted:
		if lifecycle != nil {
			updates["delete_requested_at"] = lifecycle.DeleteRequestedAt
			updates["retention_until"] = lifecycle.RetentionUntil
		}
	case model.TenantActive:
		// 恢复：清空注销时间线（frozen → active 不触碰到这些列）
		updates["delete_requested_at"] = nil
		updates["retention_until"] = nil
	}

	if err := db.Model(&model.Tenant{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	t.invalidateStatusCache(id)
	return nil
}

// ListPurgeable 注销到期待清理租户：status=deleted、保留期已过、未落墓碑。
// Unscoped 语义上等价（deleted 行不再走软删），显式声明防未来模型语义回退
func (t *tenantRepository) ListPurgeable(ctx context.Context, now time.Time) ([]model.Tenant, error) {
	tenants := make([]model.Tenant, 0)
	if err := t.db.WithContext(ctx).Unscoped().
		Where("status = ? AND retention_until IS NOT NULL AND retention_until <= ? AND purged_at IS NULL",
			model.TenantDeleted, now).
		Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

// PurgeTenantData 物理清理租户业务数据（FIX-012）：
// 1. 关系表（user_roles/user_groups/group_roles/department_users）按租户
//    维度的成员/分组/角色/部门硬删——这些表无 GORM 模型与软删语义；
// 2. 租户内四张业务表硬删（Unscoped，注销清理即销毁，不保留软删行）；
// 3. 租户行保留并落 purged_at 墓碑（code 唯一约束防止注销编码复用）。
// 整体一个事务：清理失败整体回滚，可重试
func (t *tenantRepository) PurgeTenantData(ctx context.Context, tenantID uint) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, stmt := range []string{
			`DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE tenant_id = ?)`,
			`DELETE FROM user_groups WHERE user_id IN (SELECT id FROM users WHERE tenant_id = ?)`,
			`DELETE FROM group_roles WHERE group_id IN (SELECT id FROM groups WHERE tenant_id = ?)`,
			`DELETE FROM department_users WHERE department_id IN (SELECT id FROM departments WHERE tenant_id = ?)`,
			`DELETE FROM users WHERE tenant_id = ?`,
			`DELETE FROM groups WHERE tenant_id = ?`,
			`DELETE FROM roles WHERE tenant_id = ?`,
			`DELETE FROM departments WHERE tenant_id = ?`,
		} {
			if err := tx.Exec(stmt, tenantID).Error; err != nil {
				return err
			}
		}

		// 墓碑标记：保留行本身，置 purged_at
		return tx.Model(&model.Tenant{}).Where("id = ?", tenantID).
			Update("purged_at", time.Now()).Error
	})
}
