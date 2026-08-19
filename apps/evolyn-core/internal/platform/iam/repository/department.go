package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

type departmentRepository struct {
	db  *gorm.DB
	rdb *infrastructure.RedisDB
}

func newDepartmentRepository(db *gorm.DB, rdb *infrastructure.RedisDB) DepartmentRepository {
	return &departmentRepository{
		db:  db,
		rdb: rdb,
	}
}

// withContext 以请求 ctx 打开新会话，租户过滤由 GORM Callback 自动注入；
// ctx 携带事务 session 时加入外层事务（FIX-021：成员创建与部门绑定同事务）
func (d *departmentRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, d.db)
}

func (d *departmentRepository) List(ctx context.Context) ([]model.Department, error) {
	depts := make([]model.Department, 0)
	// 排序规则：层级树展示按 parent → order 稳定输出
	if err := d.withContext(ctx).Order("parent_id").Order(`"order"`).Find(&depts).Error; err != nil {
		return nil, err
	}
	return depts, nil
}

func (d *departmentRepository) GetByID(ctx context.Context, id uint) (*model.Department, error) {
	dept := new(model.Department)
	if err := d.withContext(ctx).First(dept, id).Error; err != nil {
		return nil, err
	}
	return dept, nil
}

func (d *departmentRepository) Create(ctx context.Context, dept *model.Department) (*model.Department, error) {
	if err := d.withContext(ctx).Create(dept).Error; err != nil {
		return nil, err
	}
	return dept, nil
}

func (d *departmentRepository) Update(ctx context.Context, dept *model.Department) (*model.Department, error) {
	if err := d.withContext(ctx).Model(&model.Department{}).Where("id = ?", dept.ID).
		Select("parent_id", "name", `"order"`, "status").Updates(dept).Error; err != nil {
		return nil, err
	}
	return dept, nil
}

// Delete 删除部门（软删）；成员关联随成员侧 SetMemberDepartments 清理
func (d *departmentRepository) Delete(ctx context.Context, id uint) error {
	return d.withContext(ctx).Delete(&model.Department{}, id).Error
}

// SetMemberDepartments 整体替换成员的部门归属（多部门，对齐简道云 member.dept）
func (d *departmentRepository) SetMemberDepartments(ctx context.Context, member *model.User, departmentIDs []uint) error {
	if len(departmentIDs) == 0 {
		return d.withContext(ctx).Model(member).Association(model.DepartmentAssociation).Clear()
	}
	depts := make([]model.Department, 0, len(departmentIDs))
	for _, id := range departmentIDs {
		depts = append(depts, model.Department{ID: id})
	}
	return d.withContext(ctx).Model(member).Association(model.DepartmentAssociation).Replace(depts)
}

func (d *departmentRepository) Migrate() error {
	return d.db.AutoMigrate(&model.Department{})
}
