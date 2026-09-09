// DML 值编解码：协议 JSON 值形态 ↔ 物理列 SQL 参数/扫描值的类型收敛。
//
// 值形态契约（与 JSONB 记录出网完全一致，保证出网/权限判定/流程表达式在
// 两种存储模式下同构）：文本/单选=string、数字=float64、date="YYYY-MM-DD"、
// datetime="YYYY-MM-DD HH:MM:SS"（本地形状直存，与 JSONTime 口径一致）、
// month="YYYY-MM"、time="HH:MM"（两者原形 TEXT 直存）、单成员/单部门引用
// =string(十进制 ID)。
package storage

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	kernel "evolyn/internal/model"
)

// EncodeSQLValue 按值语义把协议 JSON 值收敛为可绑定物理列的 SQL 参数。
// nil 与空串/空数组统一收敛为 NULL（物理列可空，空值两态在读取侧还原为 nil）。
func EncodeSQLValue(kind FieldKind, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch kind {
	case KindText:
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("文本字段值必须是字符串，得到 %T", value)
		}
		if text == "" {
			return nil, nil
		}
		return text, nil
	case KindNumber:
		number, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("数字字段值必须是数值，得到 %T", value)
		}
		return number, nil
	case KindDate, KindDateTime:
		text, ok := value.(string)
		if !ok || text == "" {
			return nil, fmt.Errorf("日期字段值必须是规范形状字符串，得到 %T", value)
		}
		// 提交终审已做过形状与真实日历校验；此处仅做形状防御复核，
		// 拒绝任何绕过校验器进入动态 DML 的值。
		if !canonicalDateOrDateTime(kind, text) {
			return nil, fmt.Errorf("日期字段值 %q 不符合规范形状", text)
		}
		return text, nil
	case KindMonth, KindTime:
		text, ok := value.(string)
		if !ok || text == "" {
			return nil, fmt.Errorf("时间字段值必须是规范形状字符串，得到 %T", value)
		}
		// 同上：终审已校验（含 month 的真实日历月），此处防御复核形状，
		// 保证 TEXT 列内容恒为等宽零填充定长形状（字典序==时间序的前提）。
		if !canonicalMonthOrTime(kind, text) {
			return nil, fmt.Errorf("时间字段值 %q 不符合规范形状", text)
		}
		return text, nil
	case KindRef:
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("引用字段值必须是字符串 ID，得到 %T", value)
		}
		if text == "" {
			return nil, nil
		}
		id, err := strconv.ParseInt(text, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("引用字段值 %q 不是合法的正整数 ID", text)
		}
		return id, nil
	default:
		return nil, fmt.Errorf("未知值语义 %q", kind)
	}
}

// DecodeSQLValue 把物理列扫描值（database/sql 驱动返回形态）还原为协议
// JSON 值形态。NULL → nil（与 JSONB 记录缺键/显式 null 同出网语义）。
func DecodeSQLValue(kind FieldKind, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch kind {
	case KindText:
		switch v := value.(type) {
		case string:
			if v == "" {
				return nil, nil
			}
			return v, nil
		case []byte:
			text := string(v)
			if text == "" {
				return nil, nil
			}
			return text, nil
		default:
			return nil, fmt.Errorf("文本列扫描到未知类型 %T", value)
		}
	case KindNumber:
		switch v := value.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case []byte: // pgx 数值文本回传路径
			number, err := strconv.ParseFloat(string(v), 64)
			if err != nil {
				return nil, fmt.Errorf("数值列扫描值 %q 非法", string(v))
			}
			return number, nil
		case string:
			number, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("数值列扫描值 %q 非法", v)
			}
			return number, nil
		default:
			return nil, fmt.Errorf("数值列扫描到未知类型 %T", value)
		}
	case KindDate:
		return decodeTimeText(kind, value, "2006-01-02")
	case KindDateTime:
		return decodeTimeText(kind, value, "2006-01-02 15:04:05")
	case KindMonth, KindTime:
		// 原形 TEXT 直存：扫描值还原为字符串（空串视同未填写），形状
		// 防御复核防脏数据污染字典序前提。
		var text string
		switch v := value.(type) {
		case string:
			text = v
		case []byte:
			text = string(v)
		default:
			return nil, fmt.Errorf("%s 列扫描到未知类型 %T", kind, value)
		}
		if text == "" {
			return nil, nil
		}
		if !canonicalMonthOrTime(kind, text) {
			return nil, fmt.Errorf("%s 列扫描值 %q 不符合规范形状", kind, text)
		}
		return text, nil
	case KindRef:
		switch v := value.(type) {
		case int64:
			return strconv.FormatInt(v, 10), nil
		case uint64:
			return strconv.FormatUint(v, 10), nil
		case []byte:
			return string(v), nil
		case string:
			return v, nil
		default:
			return nil, fmt.Errorf("引用列扫描到未知类型 %T", value)
		}
	default:
		return nil, fmt.Errorf("未知值语义 %q", kind)
	}
}

// decodeTimeText 日期/时间列统一按东八区规范形状出文本（与 JSONTime 出网
// 口径一致；TIMESTAMP 列驱动返回的 wall-clock 即存储形状，Format 不做时区换算）。
func decodeTimeText(kind FieldKind, value any, layout string) (any, error) {
	var parsed time.Time
	switch v := value.(type) {
	case time.Time:
		parsed = v
	case string:
		var err error
		parsed, err = time.ParseInLocation(layout, v, kernel.CSTLocation())
		if err != nil {
			return nil, fmt.Errorf("%s 列扫描值 %q 不符合规范形状", kind, v)
		}
	case []byte:
		text := string(v)
		var err error
		parsed, err = time.ParseInLocation(layout, text, kernel.CSTLocation())
		if err != nil {
			return nil, fmt.Errorf("%s 列扫描值 %q 不符合规范形状", kind, text)
		}
	default:
		return nil, fmt.Errorf("%s 列扫描到未知类型 %T", kind, value)
	}
	text := parsed.Format(layout)
	if text == "" || strings.HasPrefix(text, "0001-") {
		return nil, nil // 零值防御：视同未填写
	}
	return text, nil
}

// canonicalDateOrDateTime 规范形状防御（真实日历校验在提交终审完成）。
func canonicalDateOrDateTime(kind FieldKind, text string) bool {
	if kind == KindDate {
		if len(text) != 10 {
			return false
		}
	} else if len(text) != 19 {
		return false
	}
	if text[4] != '-' || text[7] != '-' {
		return false
	}
	if kind == KindDateTime && text[10] != ' ' {
		return false
	}
	for _, r := range text {
		if (r < '0' || r > '9') && r != '-' && r != ' ' && r != ':' {
			return false
		}
	}
	return true
}

// canonicalMonthOrTime month（YYYY-MM，7 位）/time（HH:MM，5 位）规范形状
// 防御：等宽零填充是「字典序==时间序」的前提，任何变长或非数字形状都会
// 破坏比较语义，进入物理列前必须拒绝。
func canonicalMonthOrTime(kind FieldKind, text string) bool {
	if kind == KindMonth {
		if len(text) != 7 || text[4] != '-' {
			return false
		}
	} else if len(text) != 5 || text[2] != ':' {
		return false
	}
	for _, r := range text {
		if (r < '0' || r > '9') && r != '-' && r != ':' {
			return false
		}
	}
	// 分量范围复核：月 01-12、时 00-23、分 00-59（终审已保证，此处兜底）
	if kind == KindMonth {
		month := int(text[5]-'0')*10 + int(text[6]-'0')
		return month >= 1 && month <= 12
	}
	hour := int(text[0]-'0')*10 + int(text[1]-'0')
	minute := int(text[3]-'0')*10 + int(text[4]-'0')
	return hour <= 23 && minute <= 59
}
