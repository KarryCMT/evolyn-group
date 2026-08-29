package model

// CCRecord 抄送记录（第 10.6 章）：CC 不是审批 Task，不参与节点完成判定，
// 只读知会。落独立记录表（迁移 000051）作为「抄送我的」查询模型的最简实现
// （设计文档 10.6 章允许以查询模型最简为准选择存储形态）；追加写，禁止更新。
type CCRecord struct {
	ID uint
	// TenantID 归属租户
	TenantID uint
	// InstanceID / NodeInstanceID 归属实例与抄送节点实例
	InstanceID     uint
	NodeInstanceID uint
	// NodeKey 抄送节点 key（设计态）
	NodeKey string
	// MemberID 抄送对象成员 ID（解析时一次性快照，v1.1 定版语义同审批人）
	MemberID uint
	// DisplayName 显示名快照
	DisplayName string
}
