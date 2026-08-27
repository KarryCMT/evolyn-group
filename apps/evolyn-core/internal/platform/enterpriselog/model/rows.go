package model

import "time"

// LoginLogRow 登录日志读取行（仓储 → 服务层）：出网投影的原料。
// DisplayName 为 JOIN users/accounts 兜底的当前成员显示名——仅存量历史行
// （actor_name_snapshot 为空）使用，新写入行优先展示快照
type LoginLogRow struct {
	ID                uint
	ActorNameSnapshot string
	MemberID          uint
	Client            string
	IP                string
	Location          string
	CreatedAt         time.Time
	DisplayName       string
}

// AuditLogRow 操作日志读取行（仓储 → 服务层）：不含 before/after 原始
// 快照（受控内部审计数据不进企业日志出网链路）
type AuditLogRow struct {
	ID                uint
	MemberID          uint
	EventCode         string
	CategoryCode      string
	ActorNameSnapshot string
	Summary           string
	IP                string
	CreatedAt         time.Time
	DisplayName       string
}
