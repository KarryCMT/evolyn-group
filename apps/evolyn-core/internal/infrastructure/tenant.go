package infrastructure

import (
	"evolyn/internal/contextx"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TenantScope 显式租户过滤 Scope，供绕过 context 的路径（脚本/后台任务）使用。
// 常规请求路径由 RegisterTenantCallbacks 注册的 Callback 自动注入，Repository 禁止手写租户条件。
// 实现注意：GORM 的 Scope 在链式调用时立即执行，此刻 Statement.Schema 尚未解析，
// 不能用 LookUpField 反射守卫——否则过滤条件被静默吞掉，查询退化为全表
// （曾致配额统计数到全平台成员，任何租户开通被误杀）
func TenantScope(tenantID uint) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if tenantID == 0 {
			return db
		}
		return db.Where(map[string]interface{}{"tenant_id": tenantID})
	}
}

// RegisterTenantCallbacks 注册租户 Callback（架构文档 26.3）：
//   - Create：dest 的 TenantID 为零时从 Statement.Context 注入（未注入则落列默认值默认租户）；
//   - Query/Update/Delete：Statement.Context 携带租户时自动追加 tenant_id 过滤。
//
// 现状（P0）service 层尚未线程化 context，无租户上下文时 Callback 无副作用；
// M1 ctx 线程化后即全面生效，两条数据路径（GORM/pgx）都不允许业务侧手写租户条件
func RegisterTenantCallbacks(db *gorm.DB) error {
	// 操作者字段与租户字段都由请求上下文注入。即使业务层构造了模型，也不能
	// 伪造当前登录账号；系统任务没有 Actor 时则保留数据库 NULL。
	if err := db.Callback().Create().Before("gorm:create").Register("evolyn:actor_create", actorCreateCallback); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("gorm:update").Register("evolyn:actor_update", actorUpdateCallback); err != nil {
		return err
	}
	if err := db.Callback().Create().Before("gorm:create").Register("evolyn:tenant_create", tenantCreateCallback); err != nil {
		return err
	}
	if err := db.Callback().Query().Before("gorm:query").Register("evolyn:tenant_query", tenantConditionCallback); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("gorm:update").Register("evolyn:tenant_update", tenantConditionCallback); err != nil {
		return err
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register("evolyn:tenant_delete", tenantConditionCallback); err != nil {
		return err
	}
	return nil
}

// actorCreateCallback 以平台账号作为统一操作者身份，同时写入创建人与更新人。
// GORM 的 SetColumn 对结构体和 map 更新均生效；未认证路径不触碰字段，保留 NULL。
func actorCreateCallback(tx *gorm.DB) {
	if tx.Statement.Schema == nil || tx.Statement.Schema.LookUpField("CreatorID") == nil {
		return
	}
	actor, ok := contextx.ActorFromContext(tx.Statement.Context)
	if !ok || actor.AccountID == 0 {
		return
	}
	accountID := actor.AccountID
	tx.Statement.SetColumn("CreatorID", &accountID)
	if tx.Statement.Schema.LookUpField("UpdaterID") != nil {
		tx.Statement.SetColumn("UpdaterID", &accountID)
	}
}

// actorUpdateCallback 仅刷新最后更新账号。创建人不可由普通更新路径改变。
func actorUpdateCallback(tx *gorm.DB) {
	if tx.Statement.Schema == nil || tx.Statement.Schema.LookUpField("UpdaterID") == nil {
		return
	}
	actor, ok := contextx.ActorFromContext(tx.Statement.Context)
	if !ok || actor.AccountID == 0 {
		return
	}
	accountID := actor.AccountID
	tx.Statement.SetColumn("UpdaterID", &accountID)
}

func tenantCreateCallback(tx *gorm.DB) {
	if tx.Statement.Schema == nil || tx.Statement.Schema.LookUpField("TenantID") == nil {
		return
	}

	tenantID, ok := contextx.TenantIDFromContext(tx.Statement.Context)
	if !ok {
		return
	}

	setTenantID(tx.Statement.ReflectValue, tenantID)
}

func tenantConditionCallback(tx *gorm.DB) {
	tenantID, ok := contextx.TenantIDFromContext(tx.Statement.Context)
	if !ok {
		return
	}

	if tenantID == 0 || tx.Statement.Schema == nil {
		return
	}
	if tx.Statement.Schema.LookUpField("TenantID") == nil {
		return
	}

	tx.Statement.AddClause(clause.Where{Exprs: []clause.Expression{
		clause.Eq{Column: clause.Column{Name: "tenant_id"}, Value: tenantID},
	}})
}

// setTenantID 将零值 TenantID 填充为当前租户，支持批量 dest（切片）与结构体
func setTenantID(rv reflect.Value, tenantID uint) {
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			setTenantID(rv.Index(i), tenantID)
		}
	case reflect.Struct:
		field := rv.FieldByName("TenantID")
		if field.IsValid() && field.CanSet() && field.Kind() == reflect.Uint && field.Uint() == 0 {
			field.SetUint(uint64(tenantID))
		}
	}
}
