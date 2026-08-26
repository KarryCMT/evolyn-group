package model

// ---- 出网 DTO（文档 6.1/6.2/6.3）----

// ProductCenterView 产品中心卡片列表（GET /tenant-products 响应）
type ProductCenterView struct {
	Items []ProductCard `json:"items"`
}

// ProductCard 产品中心卡片：版本信息来自 edition 域事实源（不在产品配置
// 冗余存储）；eligibleMemberCount 按真实有效成员计算（文档 2 结论 3）
type ProductCard struct {
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Icon        string          `json:"icon"`
	Enabled     bool            `json:"enabled"`
	Revision    int64           `json:"revision"`
	Edition     ProductEdition  `json:"edition"`
	AccessScope AccessScopeView `json:"accessScope"`
	EntryPath   string          `json:"entryPath"`
}

// ProductEdition 卡片「当前版本」投影：订阅的套餐与状态（读时投影语义
// 同 GET /editions/current，expired 表示到期未降级窗口）
type ProductEdition struct {
	PlanCode string `json:"planCode"`
	PlanName string `json:"planName"`
	Status   string `json:"status"`
}

// AccessScopeView 可用范围视图：departmentIds/memberIds/selections 均为
// 过滤悬挂引用后的当前有效选择（部门删除/停用、成员离职/禁用时读取侧
// 直接丢弃，不放大范围也不报错，文档 5.5）
type AccessScopeView struct {
	Mode                string           `json:"mode"`
	EligibleMemberCount int64            `json:"eligibleMemberCount"`
	DepartmentIds       []uint           `json:"departmentIds"`
	MemberIds           []uint           `json:"memberIds"`
	Selections          []ScopeSelection `json:"selections"`
}

// ScopeSelection 选择项展示：type 取 department / member，label 为当前名称
type ScopeSelection struct {
	Type  string `json:"type"`
	ID    uint   `json:"id"`
	Label string `json:"label"`
}

// ---- 入网请求（文档 6.2/6.3）----

// UpdateEnabledRequest 启停请求：revision 为读取卡片时拿到的乐观锁版本。
// Enabled 不加 required 约束（false 是合法值，required 会把零值误判为缺失）
type UpdateEnabledRequest struct {
	Enabled  bool  `json:"enabled"`
	Revision int64 `json:"revision"`
}

// UpdateAccessScopeRequest 范围全量替换请求：mode=all 时 ID 清单必须为空
// 或省略；mode=partial 时两者合计至少一个（服务层校验）
type UpdateAccessScopeRequest struct {
	Mode          string `json:"mode" binding:"required"`
	DepartmentIds []uint `json:"departmentIds"`
	MemberIds     []uint `json:"memberIds"`
	Revision      int64  `json:"revision"`
}
