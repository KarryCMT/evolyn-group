// Package model 表单资产域数据模型（后端契约 §1）。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	kernel "evolyn/internal/model"
)

// FormType 表单资产类型：标准表单只承载字段与数据，流程表单额外开放流程
// 设计能力。ADR-011 起类型可经 form-actions:switch-type 动作切换
// （standard↔workflow），切换后原类型流程数据保留；切换裁决在 Service 层。
type FormType string

// CurrentProtocolVersion 当前表单保存协议版本；v12 将字段默认值公式纳入
// content.fieldFormulas，并在发布版本冻结服务端可执行产物。
const CurrentProtocolVersion = 12

// FieldIdentityProtocolVersion 字段身份协议（v8）的最低协议版本：该版本起
// 值字段必须携带内部不可变 fieldId（10 位小写 base32），发布后 fieldId 与
// widgetName 均冻结（只允许改 label）。更早协议的存量表单保持 JSONB 兼容，
// 不做批量迁移；新建表单固定以当前协议创建（物理存储绑定的前置条件）。
const FieldIdentityProtocolVersion = 8

// InvisibleValuePolicyVersion 不可见字段赋值管线（v6）的最低协议版本：该版本
// 起提交终审按「有效可见性 → 策略决议 → 值终审」执行；更早的发布快照保持
// 旧静态可见语义，不做批量迁移（历史记录按绑定快照解释）。
const InvisibleValuePolicyVersion = 6

const (
	FormTypeStandard FormType = "standard"
	FormTypeWorkflow FormType = "workflow"
)

// Valid 判断表单类型是否属于稳定枚举，禁止未知字符串进入持久化层。
func (t FormType) Valid() bool {
	return t == FormTypeStandard || t == FormTypeWorkflow
}

// Form 表单资产（含草稿）：租户内从属于应用；草稿全文为目标保存协议文档，
// draft_revision 为草稿乐观锁口令；发布快照另存 tn_form_versions（不可变）。
type Form struct {
	ID               uint        `json:"id" gorm:"autoIncrement;primaryKey"`
	AppID            uint        `json:"appId" gorm:"not null"`                             // 所属应用（同租户，Service 层校验）
	Code             string      `json:"code" gorm:"size:64;not null"`                      // form_ 前缀稳定公开编码（路由/API 使用）
	Name             string      `json:"name" gorm:"size:128;not null"`                     // 表单名称（不进入协议 content）
	FormType         FormType    `json:"formType" gorm:"size:16;not null;default:standard"` // 表单类型（ADR-011 起可经动作切换）
	DraftContent     JSONContent `json:"draft" gorm:"type:jsonb;not null"`
	DraftRevision    int64       `json:"draftRevision" gorm:"not null;default:1"`   // 草稿乐观锁：保存条件递增
	ProtocolVersion  int         `json:"protocolVersion" gorm:"not null;default:7"` // 协议版本外部承载（文档内无版本字段；default 仅 ORM 声明，写入方恒显式赋值）
	LatestVersionID  *uint       `json:"latestVersionId"`                           // 最新发布版本；NULL=从未发布
	PublishedVersion int         `json:"publishedVersion" gorm:"not null;default:0"`
	CreatorMemberID  uint        `json:"creatorMemberId" gorm:"not null"`

	kernel.TenantBaseModel
}

func (*Form) TableName() string { return "tn_forms" }

// FormVersion 不可变发布快照：发布事务内一次写入，之后不存在更新路径；
// schema_revision 即行 id（出网字符串），提交校验与记录归属的双口令之一。
type FormVersion struct {
	ID             uint        `json:"id" gorm:"autoIncrement;primaryKey"`
	FormID         uint        `json:"formId" gorm:"not null"`
	VersionNo      int         `json:"versionNo" gorm:"not null"` // 表单内递增发布号 1,2,3…
	SchemaRevision int64       `json:"schemaRevision" gorm:"not null;default:0"`
	Content        JSONContent `json:"content" gorm:"type:jsonb;not null"`
	FieldKeys      JSONContent `json:"fieldKeys" gorm:"type:jsonb;not null"` // 顶层字段键有序数组（提交未知键快速拒绝）
	// FieldMappings 为发布时冻结的逻辑字段→JSONB/物理列映射；查询编译器只能消费此快照。
	FieldMappings JSONContent `json:"fieldMappings" gorm:"type:jsonb;not null"`
	// CompiledSubmitRules 与发布版本同事务冻结；提交绝不读取草稿重新解释规则。
	CompiledSubmitRules JSONContent `json:"-" gorm:"type:jsonb;not null;default:'{}'"`
	// CompiledFieldFormulas 是字段派生值的权威执行产物；客户端计算仅用于即时预览。
	CompiledFieldFormulas JSONContent     `json:"-" gorm:"type:jsonb;not null;default:'{}'"`
	ProtocolVersion       int             `json:"protocolVersion" gorm:"not null;default:7"`
	PublishedByMemberID   uint            `json:"publishedByMemberId" gorm:"not null"`
	PublishedAt           kernel.JSONTime `json:"publishedAt"`

	TenantID  uint            `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

func (*FormVersion) TableName() string { return "tn_form_versions" }

// FormRecord 记录提交：追加写；form_version_id 固定受理时所依据的版本
// （历史版本合法）。存量记录的业务值在 values JSONB（键=widgetName）；新
// 记录（physical 存储）values 恒为 NULL，业务值只存在于对应 tn_fd_* 物理
// 表行——一份业务事实源，禁止双写（物理表存储方案 §5.1）。
type FormRecord struct {
	// WorkflowInstanceNo 独立只读系统字段，普通表单为空；不混入用户 values。
	WorkflowInstanceNo string `json:"workflowInstanceNo" gorm:"size:40;not null;default:''"`
	// WorkflowStatus/WorkflowUpdatedAt 流程实例状态投影（事实源 wf_instance.status）：
	// 随实例状态变更同事务刷新；普通表单恒 NONE/NULL。physical 表另预置同名列
	// 作高频筛选投影。
	WorkflowStatus    string           `json:"workflowStatus" gorm:"size:16;not null;default:'NONE'"`
	WorkflowUpdatedAt *kernel.JSONTime `json:"workflowUpdatedAt"`

	ID                  uint        `json:"id" gorm:"autoIncrement;primaryKey"`
	FormID              uint        `json:"formId" gorm:"not null"`
	FormVersionID       uint        `json:"formVersionId" gorm:"not null"`
	DataOpID            *string     `json:"dataOpId" gorm:"size:36"`  // 客户端提交幂等键；历史记录允许 NULL
	MenuCode            *string     `json:"menuCode" gorm:"size:64"`  // 提交入口菜单编码快照；预览直提允许 NULL
	Values              JSONContent `json:"values" gorm:"type:jsonb"` // 仅存量历史记录；新记录恒 NULL
	SubmittedByMemberID uint        `json:"submittedByMemberId" gorm:"not null"`
	// SubmittedByName 提交人展示名快照：提交时按租户内昵称固化，成员改名/
	// 退出后历史展示不失真（与企业日志 actor_name_snapshot 口径一致）。
	SubmittedByName string          `json:"submittedByName" gorm:"size:100"`
	SubmittedAt     kernel.JSONTime `json:"submittedAt"`
	// UpdatedAt 最后写回时间：提交时=提交时间，审批编辑写回 values 时刷新。
	UpdatedAt kernel.JSONTime `json:"updatedAt"`
	// UpdatedByMemberID/UpdatedByName 最后写人人（000072）：提交时=提交人，
	// 审批编辑/发起人修改写回时刷新为操作人快照；系统自动路径（无操作人
	// 上下文）保持原值。展示名快照与提交人同口径：改名/退出后不失真。
	UpdatedByMemberID uint   `json:"updatedByMemberId"`
	UpdatedByName     string `json:"updatedByName" gorm:"size:100"`

	TenantID  uint            `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

func (*FormRecord) TableName() string { return "tn_form_records" }

// FormSerialCounter 是流水号的服务端计数事实源。NextValue 始终表示下一个可分配值；
// 周期键由发号服务按提交时刻生成，避免浏览器时钟或记录删除影响连续性。
type FormSerialCounter struct {
	ID        uint   `gorm:"autoIncrement;primaryKey"`
	TenantID  uint   `gorm:"not null;uniqueIndex:uq_form_serial_counter_scope,priority:1"`
	FormID    uint   `gorm:"not null;uniqueIndex:uq_form_serial_counter_scope,priority:2"`
	FieldID   string `gorm:"size:10;not null;uniqueIndex:uq_form_serial_counter_scope,priority:3"`
	CycleKey  string `gorm:"size:16;not null;uniqueIndex:uq_form_serial_counter_scope,priority:4"`
	NextValue int64  `gorm:"not null"`

	// 计数器是技术状态表，不承载操作者与软删除；保留与 SQL migration 一致的三列。
	CreatedAt kernel.JSONTime `json:"createdAt"`
	UpdatedAt kernel.JSONTime `json:"updatedAt"`
}

func (*FormSerialCounter) TableName() string { return "tn_form_serial_counters" }

// JSONContent JSONB 原文载体：保存协议要求「未编辑属性不丢失」，因此草稿/快照
// 一律原样字节存取（校验在 Service 层完成），不经 map 往返避免键序与空值失真。
type JSONContent json.RawMessage

// Value 实现 driver.Valuer：nil（未设置=physical 记录不写 JSONB 值）落 NULL，
// 空切片落 '{}'（存量空对象语义），其余以字符串形态交给 pgx 写入 jsonb 列
func (j JSONContent) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

// Scan 实现 sql.Scanner：接受 pgx 返回的 []byte / string
func (j *JSONContent) Scan(value interface{}) error {
	switch data := value.(type) {
	case nil:
		*j = JSONContent("null")
	case []byte:
		*j = append((*j)[:0], data...)
	case string:
		*j = JSONContent(data)
	default:
		return fmt.Errorf("form: cannot scan JSONContent from %T", value)
	}
	return nil
}

// MarshalJSON 原样出网（json.RawMessage 语义，避免二次转义）
func (j JSONContent) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON 原样收网
func (j *JSONContent) UnmarshalJSON(data []byte) error {
	*j = append((*j)[:0], data...)
	return nil
}
