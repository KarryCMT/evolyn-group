package repository

// 流程变量仓储（000053，Phase 7）：引擎内核 VariableRepository SPI 的
// GORM 适配。值以 JSONB 单值承载（V1 冻结标量值域），(instance_id, var_key)
// 唯一——覆盖写语义，同事务 upsert（ON CONFLICT DO UPDATE）。

import (
	"context"
	"encoding/json"
	"fmt"

	enginemodel "evolyn/internal/engine/workflow/model"
	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/workflow/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type runtimeVariables struct{ *runtimeRepository }

// NewVariableRepository 构造变量仓储（实现引擎 VariableRepository SPI +
// 开发态 AutoMigrate，随其他运行态仓储装配注入 Runtime）。
func NewVariableRepository(base *gorm.DB) *runtimeVariables {
	return &runtimeVariables{&runtimeRepository{base: base}}
}

func (r *runtimeVariables) Migrate() error {
	return r.base.AutoMigrate(&model.WfVariable{})
}

func (r *runtimeVariables) SaveVariable(ctx context.Context, variable *enginemodel.Variable) error {
	value, err := json.Marshal(variable.Value)
	if err != nil {
		return fmt.Errorf("variable %s: %w", variable.Key, err)
	}
	// TenantID 留零由租户 Callback 从 ctx 注入（Worker 执行路径已按 Job
	// 租户上下文化，见 worker 租户上下文修复）
	row := model.WfVariable{
		InstanceID: variable.InstanceID,
		VarKey:     variable.Key,
		VarType:    string(variable.ValueType),
		VarValue:   value,
	}
	// upsert：同键覆盖（表达式环境与库内状态同事务一致）
	return infrastructure.ResolveDB(ctx, r.base).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "instance_id"}, {Name: "var_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"var_type", "var_value", "updated_at"}),
		}).
		Create(&row).Error
}

func (r *runtimeVariables) ListVariablesByInstance(ctx context.Context, instanceID uint) ([]enginemodel.Variable, error) {
	var rows []model.WfVariable
	if err := infrastructure.ResolveDB(ctx, r.base).
		Where("instance_id = ?", instanceID).
		Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	vars := make([]enginemodel.Variable, 0, len(rows))
	for i := range rows {
		var value any
		if err := json.Unmarshal(rows[i].VarValue, &value); err != nil {
			return nil, fmt.Errorf("variable %s: %w", rows[i].VarKey, err)
		}
		vars = append(vars, enginemodel.Variable{
			InstanceID: rows[i].InstanceID,
			Key:        rows[i].VarKey,
			ValueType:  enginemodel.VariableType(rows[i].VarType),
			Value:      value,
		})
	}
	return vars, nil
}
