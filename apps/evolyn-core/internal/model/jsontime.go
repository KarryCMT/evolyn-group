package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JSONTime API 出网时间的统一载体：定死为秒级 "2006-01-02 15:04:05"（对标简道云展示口径）。
// 背景：原生 time.Time 默认序列化为 RFC 3339Nano，微秒位数与时区偏移随数据库精度
// 和部署环境漂移，没有契约层面的稳定性；包装为独立类型后在序列化边界统一格式。
// GORM 侧经 Value/Scan 透明转换（底层类型即 time.Time，可识别为 Time DataType），
// CreatedAt/UpdatedAt 的自动填充与查询回填不受影响。
type JSONTime time.Time

// jsonTimeLayout 出网/入参统一格式：秒级、空格分隔、不带时区偏移
const jsonTimeLayout = "2006-01-02 15:04:05"

// cstLocation 展示时区固定东八区，与 postgres DSN 的 TimeZone=Asia/Shanghai 对齐。
// 出网字符串不带偏移量，必须先归一到固定时区，避免随进程时区漂移；
// 精简容器可能缺少 tzdata，加载失败时退化为固定 +08:00 偏移
var cstLocation = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return loc
	}
	return time.FixedZone("CST", 8*60*60)
}()

// Time 还原为标准 time.Time；业务侧比较/运算统一经本方法取值
func (t JSONTime) Time() time.Time { return time.Time(t) }

// CSTLocation 展示时区（东八区）访问入口：供日期类入参解析等需要与
// JSONTime 出网口径对齐的场景复用，避免各域重复实现加载/回退逻辑
func CSTLocation() *time.Location { return cstLocation }

// IsZero 是否零值
func (t JSONTime) IsZero() bool { return time.Time(t).IsZero() }

// MarshalJSON 出网统一走秒级东八区格式；零值输出空串，避免 "0001-01-01 00:00:00" 泄漏
func (t JSONTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return []byte(`""`), nil
	}
	return json.Marshal(tt.In(cstLocation).Format(jsonTimeLayout))
}

// UnmarshalJSON 入参兼容统一格式与 RFC 3339（秒/纳秒精度）；无偏移字面量按东八区解释
func (t *JSONTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if s == "" || s == "null" {
		*t = JSONTime{}
		return nil
	}
	tt, err := parseJSONTime(s)
	if err != nil {
		return err
	}
	*t = JSONTime(tt)
	return nil
}

// Value 实现 driver.Valuer：写库按原生 time.Time 交给驱动
func (t JSONTime) Value() (driver.Value, error) { return time.Time(t), nil }

// Scan 实现 sql.Scanner：time.Time 是 pgx 对 timestamptz 的返回值，
// 也是 GORM 注入 CreatedAt/UpdatedAt 自动时间戳时传入的类型；NULL 归零值
func (t *JSONTime) Scan(src interface{}) error {
	switch v := src.(type) {
	case nil:
		*t = JSONTime{}
	case time.Time:
		*t = JSONTime(v)
	case string:
		return t.parseString(v)
	case []byte:
		return t.parseString(string(v))
	default:
		return fmt.Errorf("cannot scan %T into model.JSONTime", src)
	}
	return nil
}

func (t *JSONTime) parseString(s string) error {
	tt, err := parseJSONTime(s)
	if err != nil {
		return err
	}
	*t = JSONTime(tt)
	return nil
}

// parseJSONTime 依次尝试统一格式（东八区）与 RFC 3339（自带偏移）
func parseJSONTime(s string) (time.Time, error) {
	if tt, err := time.ParseInLocation(jsonTimeLayout, s, cstLocation); err == nil {
		return tt, nil
	}
	if tt, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return tt, nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q, expect %q or RFC3339", s, jsonTimeLayout)
}
