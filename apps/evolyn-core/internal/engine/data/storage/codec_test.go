// DML 值编解码单测（物理表存储方案 §9）：协议 JSON 值形态 ↔ 物理列参数的
// 类型收敛与防御边界——空值两态、引用字段字符串 ID、日期形状守卫。
package storage

import (
	"reflect"
	"testing"
	"time"

	kernel "evolyn/internal/model"
)

// Encode：各 kind 的合法值收敛为可绑定参数；nil 与空串统一 NULL。
func TestEncodeSQLValue(t *testing.T) {
	cases := []struct {
		kind  FieldKind
		value any
		want  any
	}{
		{KindText, "hello", "hello"},
		{KindText, "", nil},
		{KindText, nil, nil},
		{KindNumber, float64(88.5), float64(88.5)},
		{KindDate, "2026-09-04", "2026-09-04"},
		{KindDateTime, "2026-09-04 12:30:00", "2026-09-04 12:30:00"},
		{KindRef, "42", int64(42)},
		{KindRef, "", nil},
	}
	for _, testCase := range cases {
		got, err := EncodeSQLValue(testCase.kind, testCase.value)
		if err != nil {
			t.Fatalf("encode %v(%v): %v", testCase.kind, testCase.value, err)
		}
		if !reflect.DeepEqual(got, testCase.want) {
			t.Fatalf("encode %v(%v) = %v, want %v", testCase.kind, testCase.value, got, testCase.want)
		}
	}
}

// Encode 防御：类型不符与非法日期形状必须拒绝（绕过校验器的值不得进入
// 动态 DML，方案 §16.7）。
func TestEncodeSQLValueRejectsMalformed(t *testing.T) {
	cases := []struct {
		kind  FieldKind
		value any
	}{
		{KindText, 42},
		{KindNumber, "88"},
		{KindNumber, true},
		{KindDate, "2026/09/04"},
		{KindDate, "2026-9-4"},
		{KindDateTime, "2026-09-04T12:30:00"},
		{KindDateTime, "2026-09-04 12:30"},
		{KindRef, "not-a-number"},
		{KindRef, 42},
	}
	for _, testCase := range cases {
		if _, err := EncodeSQLValue(testCase.kind, testCase.value); err == nil {
			t.Fatalf("encode %v(%v) must reject", testCase.kind, testCase.value)
		}
	}
}

// Decode：NULL → nil（出网语义与 JSONB 缺键一致）；驱动返回形态按 kind
// 还原为协议值形态（NUMERIC 文本回传、DATE/TIMESTAMP time.Time、ref 数字）。
func TestDecodeSQLValue(t *testing.T) {
	if got, err := DecodeSQLValue(KindText, nil); err != nil || got != nil {
		t.Fatalf("decode nil = %v, %v", got, err)
	}
	if got, err := DecodeSQLValue(KindText, ""); err != nil || got != nil {
		t.Fatalf("empty text decodes to nil, got %v, %v", got, err)
	}
	if got, err := DecodeSQLValue(KindText, "hello"); err != nil || got != "hello" {
		t.Fatalf("text decode: %v, %v", got, err)
	}
	if got, err := DecodeSQLValue(KindNumber, float64(12)); err != nil || got != float64(12) {
		t.Fatalf("number decode: %v, %v", got, err)
	}
	// pgx NUMERIC 常以文本回传
	if got, err := DecodeSQLValue(KindNumber, []byte("12.5")); err != nil || got != float64(12.5) {
		t.Fatalf("numeric text decode: %v, %v", got, err)
	}
	if got, err := DecodeSQLValue(KindRef, int64(42)); err != nil || got != "42" {
		t.Fatalf("ref decode: %v, %v", got, err)
	}
}

// Decode 日期/时间：东八区规范形状输出（date=10 位，datetime=19 位），
// 零值防御回落 nil。
func TestDecodeSQLValueDateTime(t *testing.T) {
	date := time.Date(2026, 9, 4, 0, 0, 0, 0, kernel.CSTLocation())
	if got, err := DecodeSQLValue(KindDate, date); err != nil || got != "2026-09-04" {
		t.Fatalf("date decode: %v, %v", got, err)
	}
	stamp := time.Date(2026, 9, 4, 12, 30, 0, 0, kernel.CSTLocation())
	if got, err := DecodeSQLValue(KindDateTime, stamp); err != nil || got != "2026-09-04 12:30:00" {
		t.Fatalf("datetime decode: %v, %v", got, err)
	}
	if got, err := DecodeSQLValue(KindDate, "2026-09-04"); err != nil || got != "2026-09-04" {
		t.Fatalf("date text decode: %v, %v", got, err)
	}
	if got, err := DecodeSQLValue(KindDate, time.Time{}); err != nil || got != nil {
		t.Fatalf("zero time decodes to nil, got %v, %v", got, err)
	}
}

// KindOf 支持矩阵：首期白名单内放行、数组类与 month/time 拒绝（方案 §4.3）。
func TestKindOfSupportMatrix(t *testing.T) {
	supported := []struct{ widgetType, format string }{
		{"text", ""}, {"textarea", ""}, {"radiogroup", ""}, {"combo", ""},
		{"number", ""}, {"datetime", "date"}, {"datetime", "datetime"},
		{"user", ""}, {"dept", ""},
	}
	for _, testCase := range supported {
		if _, ok := KindOf(testCase.widgetType, testCase.format); !ok {
			t.Fatalf("%s(%s) must be supported", testCase.widgetType, testCase.format)
		}
	}
	rejected := []struct{ widgetType, format string }{
		{"checkboxgroup", ""}, {"combocheck", ""}, {"usergroup", ""}, {"deptgroup", ""},
		{"datetime", "month"}, {"datetime", "time"}, {"datetime", ""},
		{"image", ""}, {"upload", ""}, {"address", ""}, {"location", ""},
		{"signature", ""}, {"sn", ""}, {"richtext", ""},
	}
	for _, testCase := range rejected {
		if _, ok := KindOf(testCase.widgetType, testCase.format); ok {
			t.Fatalf("%s(%s) must be rejected", testCase.widgetType, testCase.format)
		}
	}
	if kind, _ := KindOf("user", ""); kind != KindRef {
		t.Fatal("user maps to ref")
	}
	if kind, _ := KindOf("datetime", "date"); kind != KindDate {
		t.Fatal("datetime/date maps to date")
	}
}
