package infrastructure

import (
	"reflect"

	"evolyn/pkg/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TenantScope 显式租户过滤 Scope，供绕过 context 的路径（脚本/后台任务）使用。
// 常规请求路径由 RegisterTenantCallbacks 注册的 Callback 自动注入，Repository 禁止手写租户条件
func TenantScope(tenantID uint) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return applyTenantCondition(db, tenantID)
	}
}

func applyTenantCondition(db *gorm.DB, tenantID uint) *gorm.DB {
	if tenantID == 0 || db.Statement.Schema == nil {
		return db
	}
	if db.Statement.Schema.LookUpField("TenantID") == nil {
		return db
	}
	// 独立函数名便于在 SQL 日志中辨识租户过滤来源
	return db.Where(map[string]interface{}{"tenant_id": tenantID})
}

// RegisterTenantCallbacks 注册租户 Callback（架构文档 26.3）：
//   - Create：dest 的 TenantID 为零时从 Statement.Context 注入（未注入则落列默认值默认租户）；
//   - Query/Update/Delete：Statement.Context 携带租户时自动追加 tenant_id 过滤。
//
// 现状（P0）service 层尚未线程化 context，无租户上下文时 Callback 无副作用；
// M1 ctx 线程化后即全面生效，两条数据路径（GORM/pgx）都不允许业务侧手写租户条件
func RegisterTenantCallbacks(db *gorm.DB) error {
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

func tenantCreateCallback(tx *gorm.DB) {
	if tx.Statement.Schema == nil || tx.Statement.Schema.LookUpField("TenantID") == nil {
		return
	}

	tenantID, ok := common.TenantIDFromContext(tx.Statement.Context)
	if !ok {
		return
	}

	setTenantID(tx.Statement.ReflectValue, tenantID)
}

func tenantConditionCallback(tx *gorm.DB) {
	tenantID, ok := common.TenantIDFromContext(tx.Statement.Context)
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
