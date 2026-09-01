package model

import (
	"time"
)

// ProductLogRow 产品日志读取行（仓储 → 服务层）：出网投影的原料。
// DisplayName 为 JOIN tn_users/accounts 兜底的当前成员显示名——仅存量历史
// 行（actor_name_snapshot 为空）使用；不含 before/after 原始快照与请求
// 元数据（受控内部审计数据不进产品日志出网链路）
type ProductLogRow struct {
	ID                      uint
	MemberID                uint
	EventCode               string
	CategoryCode            string
	ActorNameSnapshot       string
	TargetNameSnapshot      string
	Summary                 string
	ApplicationNameSnapshot string
	IP                      string
	CreatedAt               time.Time
	DisplayName             string
}
