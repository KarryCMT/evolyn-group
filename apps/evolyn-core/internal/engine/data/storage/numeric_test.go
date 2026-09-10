// 数值字段族（decimal/money/percent，Phase 4）物理存储单测：KindDecimal 的
// 值语义分派、NUMERIC(p,s) 精度修饰（ColumnSpec/DDL 类型/Diff 类型冲突）与
// DML 编解码 canonical decimal string 收敛。
package storage

import (
	"reflect"
	"testing"
)

// KindOf：数值字段族三类型映射 KindDecimal；发布白名单开放后不得退回 JSONB。
func TestKindOfNumericFamily(t *testing.T) {
	for _, widgetType := range []string{"decimal", "money", "percent"} {
		kind, ok := KindOf(widgetType, "")
		if !ok || kind != KindDecimal {
			t.Fatalf("%s must map to KindDecimal, got %v(%v)", widgetType, kind, ok)
		}
	}
	if ColumnTypeOf(KindDecimal) != ColumnTypeNumeric {
		t.Fatal("KindDecimal maps to NUMERIC")
	}
}

// ColumnDDLType：decimal 族携带 NUMERIC(p,s) 修饰；其余语义为裸类型
// （存量 number 列保持裸 NUMERIC，不受 Phase 4 影响）。
func TestColumnDDLType(t *testing.T) {
	decimal := ColumnSpec{Kind: KindDecimal, Type: ColumnTypeNumeric, Precision: 20, Scale: 6}
	if got := decimal.ColumnDDLType(); got != "NUMERIC(20,6)" {
		t.Fatalf("ColumnDDLType = %q, want NUMERIC(20,6)", got)
	}
	number := ColumnSpec{Kind: KindNumber, Type: ColumnTypeNumeric}
	if got := number.ColumnDDLType(); got != "NUMERIC" {
		t.Fatalf("number ColumnDDLType = %q, want NUMERIC", got)
	}
	// 修饰为零值时（防御）不产生悬空括号——由 Validate 在模型层先行拒绝。
	if err := (&StorageModel{
		TableName: "tn_fd_01ab_k7q2m8",
		Columns:   []ColumnSpec{{FieldID: "aaaaaaaaaa", WidgetName: "w", WidgetType: "decimal", Kind: KindDecimal, Type: ColumnTypeNumeric}},
	}).Validate(); err == nil {
		t.Fatal("KindDecimal without modifiers must fail Validate")
	}
}

// Validate：精度护栏 1–40 / 0–s≤p；非 decimal 语义不得携带修饰。
func TestValidateNumericModifiers(t *testing.T) {
	valid := func(precision, scale int16) error {
		model := &StorageModel{
			TableName: "tn_fd_01ab_k7q2m8",
			Columns: []ColumnSpec{{
				FieldID: "aaaaaaaaaa", WidgetName: "w", WidgetType: "money",
				Kind: KindDecimal, Type: ColumnTypeNumeric, Precision: precision, Scale: scale,
			}},
		}
		return model.Validate()
	}
	if err := valid(20, 2); err != nil {
		t.Fatalf("NUMERIC(20,2) must validate: %v", err)
	}
	if err := valid(1, 0); err != nil {
		t.Fatalf("NUMERIC(1,0) must validate: %v", err)
	}
	for _, bad := range [][2]int16{{0, 0}, {41, 2}, {20, 21}, {5, 18}} {
		if err := valid(bad[0], bad[1]); err == nil {
			t.Fatalf("NUMERIC(%d,%d) must be rejected", bad[0], bad[1])
		}
	}
	// 非 decimal 语义携带修饰必须拒绝（防模型漂移）。
	err := (&StorageModel{
		TableName: "tn_fd_01ab_k7q2m8",
		Columns: []ColumnSpec{{
			FieldID: "aaaaaaaaaa", WidgetName: "w", WidgetType: "number",
			Kind: KindNumber, Type: ColumnTypeNumeric, Precision: 10, Scale: 2,
		}},
	}).Validate()
	if err == nil {
		t.Fatal("KindNumber with modifiers must be rejected")
	}
}

// Diff：精度修饰属于列类型的一部分——收窄或放大都按类型变更拒绝，
// 避免静默舍入历史值（FORM_STORAGE_TYPE_CHANGE_UNSUPPORTED）。
func TestDiffNumericPrecisionConflict(t *testing.T) {
	column := func(precision, scale int16) ColumnSpec {
		return ColumnSpec{
			FieldID: "aaaaaaaaaa", WidgetName: "w", WidgetType: "money",
			Kind: KindDecimal, Type: ColumnTypeNumeric, Precision: precision, Scale: scale,
		}
	}
	applied := &StorageModel{TableName: "tn_fd_01ab_k7q2m8", Columns: []ColumnSpec{column(20, 2)}}
	for _, next := range []ColumnSpec{column(20, 4), column(22, 2)} {
		target := &StorageModel{TableName: "tn_fd_01ab_k7q2m8", Columns: []ColumnSpec{next}}
		result, err := Diff(applied, target)
		if err != nil {
			t.Fatalf("diff: %v", err)
		}
		if result.TypeConflict == nil {
			t.Fatalf("modifier change %v must be a type conflict", next)
		}
	}
	// 完全一致的修饰 → 无冲突、无动作（同步发布）。
	target := &StorageModel{TableName: "tn_fd_01ab_k7q2m8", Columns: []ColumnSpec{column(20, 2)}}
	result, err := Diff(applied, target)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if result.TypeConflict != nil || !result.Plan.Empty {
		t.Fatal("identical modifiers must not conflict or plan")
	}
}

// Encode：decimal 族只接受 decimal string（字符串直绑，驱动转 NUMERIC，
// 不经 float）；空串/nil 收敛 NULL；非法形状与 number 值拒绝。
func TestEncodeSQLValueNumericFamily(t *testing.T) {
	cases := []struct {
		value any
		want  any
	}{
		{"123.45", "123.45"},
		{"-0.5", "-0.5"},
		{"0", "0"},
		{"", nil},
		{nil, nil},
	}
	for _, testCase := range cases {
		got, err := EncodeSQLValue(KindDecimal, testCase.value)
		if err != nil {
			t.Fatalf("encode decimal %v: %v", testCase.value, err)
		}
		if !reflect.DeepEqual(got, testCase.want) {
			t.Fatalf("encode decimal %v = %v, want %v", testCase.value, got, testCase.want)
		}
	}
	for _, bad := range []any{88.5, true, "1e3", "+1", ".5", "1.2.3", " 1"} {
		if _, err := EncodeSQLValue(KindDecimal, bad); err == nil {
			t.Fatalf("encode decimal %v must be rejected", bad)
		}
	}
}

// Decode：NUMERIC 列扫描值还原 canonical decimal string（§49：去尾随零、
// "-0" 收敛 "0"）；NULL → nil（与 JSONB 缺键同出网语义）。
func TestDecodeSQLValueNumericFamily(t *testing.T) {
	cases := []struct {
		value any
		want  any
	}{
		{[]byte("123.45"), "123.45"},
		{[]byte("1.50"), "1.5"},
		{[]byte("0.00"), "0"},
		{[]byte("-0.00"), "0"},
		{[]byte("7.500000"), "7.5"},
		{[]byte("-12"), "-12"},
		{"33.30", "33.3"},
		{int64(5), "5"},
		{nil, nil},
	}
	for _, testCase := range cases {
		got, err := DecodeSQLValue(KindDecimal, testCase.value)
		if err != nil {
			t.Fatalf("decode decimal %v: %v", testCase.value, err)
		}
		if !reflect.DeepEqual(got, testCase.want) {
			t.Fatalf("decode decimal %v = %v, want %v", testCase.value, got, testCase.want)
		}
	}
	for _, bad := range []any{[]byte("1e3"), []byte("abc"), true} {
		if _, err := DecodeSQLValue(KindDecimal, bad); err == nil {
			t.Fatalf("decode decimal %v must be rejected", bad)
		}
	}
}
