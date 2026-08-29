package model

// Actor 审批参与人（审批人/候选人/会签参与人）。Resolver 在任务创建时
// 一次性解析并快照到 wf_task_actor（v1.1 定版），运行中不随组织变化重算；
// 显示名为快照值，仅用于历史展示，实时身份以成员 ID 为准。
type Actor struct {
	// MemberID 租户成员 ID（同租户有效成员，解析侧保证）
	MemberID uint
	// DisplayName 解析时快照显示名
	DisplayName string
}

// ActorRole 参与人在任务上的角色（区分审批与会签/或签集合成员，
// 抄送不是审批任务，不参与节点完成判定，第 10.6 章）。
type ActorRole string

const (
	// ActorRoleAssignee 审批参与人（会签/或签集合成员）
	ActorRoleAssignee ActorRole = "assignee"
	// ActorRoleCC 抄送对象（只读，不参与完成判定）
	ActorRoleCC ActorRole = "cc"
)
