// 控件 → 物理列类型映射（方案 §4.3 首期支持矩阵的唯一后端事实源）。
//
// 首期只开放可无损转换且筛选语义明确的顶层标量字段；数组类（多选/多成员/
// 多部门）、month/time 形状日期与尚未建模的控件一律拒绝物理发布，禁止为
// 兼容把用户业务值退回 JSONB。本矩阵与前端物理类型映射口径一致（方案
// Phase 0.2：relation 统一为 BIGINT，不使用 UUID）。
package storage

// FieldKind 字段值的物理存储语义：决定列类型、DML 编解码与查询比较方式。
type FieldKind string

const (
	// KindText 文本（text/textarea/单选项值）。
	KindText FieldKind = "text"
	// KindNumber 数字（NUMERIC，保留 precision 由值校验层负责）。
	KindNumber FieldKind = "number"
	// KindDate 日期（形状 YYYY-MM-DD）。
	KindDate FieldKind = "date"
	// KindDateTime 日期时间（形状 YYYY-MM-DD HH:MM:SS，本地时间直存，
	// 与 JSONTime 东八区出网口径一致，不做时区换算）。
	KindDateTime FieldKind = "datetime"
	// KindRef 单成员/单部门引用（BIGINT；协议值形态是字符串 ID）。
	KindRef FieldKind = "ref"
)

// ColumnType PostgreSQL 列类型（DDL 输出的稳定枚举，不带长度/精度修饰）。
type ColumnType string

const (
	ColumnTypeText      ColumnType = "TEXT"
	ColumnTypeNumeric   ColumnType = "NUMERIC"
	ColumnTypeDate      ColumnType = "DATE"
	ColumnTypeTimestamp ColumnType = "TIMESTAMP"
	ColumnTypeBigint    ColumnType = "BIGINT"
	ColumnTypeInteger   ColumnType = "INTEGER"
)

// KindOf 按控件类型（与 datetime 的 format 形状）推导物理值语义。
// ok=false 表示该控件尚不具备非 JSONB 物理模型，发布期必须拒绝
// （FORM_STORAGE_UNSUPPORTED_FIELD），不得静默改用其他类型承载。
func KindOf(widgetType, format string) (FieldKind, bool) {
	switch widgetType {
	case "text", "textarea", "radiogroup", "combo":
		return KindText, true
	case "number":
		return KindNumber, true
	case "datetime":
		// month/time 无无损标量映射：month 缺日、time 无日期，转 DATE/TIMESTAMP
		// 都会引入捏造分量，保持 JSONB 形状直到形状建模定板。
		switch format {
		case "date":
			return KindDate, true
		case "datetime":
			return KindDateTime, true
		default:
			return "", false
		}
	case "user", "dept":
		// 单成员/单部门引用：后端主键体系是 BIGINT，协议字符串 ID 编解码收敛。
		return KindRef, true
	default:
		// checkboxgroup/combocheck/usergroup/deptgroup（数组）、subform（独立子表，
		// 由子表建模分支处理）与全部未开放控件。
		return "", false
	}
}

// ColumnTypeOf 值语义 → PostgreSQL 列类型。
func ColumnTypeOf(kind FieldKind) ColumnType {
	switch kind {
	case KindText:
		return ColumnTypeText
	case KindNumber:
		return ColumnTypeNumeric
	case KindDate:
		return ColumnTypeDate
	case KindDateTime:
		return ColumnTypeTimestamp
	case KindRef:
		return ColumnTypeBigint
	default:
		return ""
	}
}

// NoColumnWidgetType 报告控件是否为「无记录值」的布局/按钮类（不进入
// 物理模型，也不进入查询白名单）。
func NoColumnWidgetType(widgetType string) bool {
	switch widgetType {
	case "separator", "button":
		return true
	default:
		return false
	}
}
