// Diff 与模型校验单测（物理表存储方案 §6/§8/§16.11）：字段演进的全部
// 计划形态——首次建表、加列、弃用合并、类型冲突拒绝、表名不可变、空计划
// 同步路径与 checksum 确定性。
package storage

import (
	"strings"
	"testing"
)

func column(fieldID, name string, kind FieldKind) ColumnSpec {
	return ColumnSpec{FieldID: fieldID, WidgetName: name, WidgetType: "text", Kind: kind, Type: ColumnTypeOf(kind)}
}

func parentModel(table string, columns ...ColumnSpec) *StorageModel {
	return &StorageModel{TableName: table, Columns: columns}
}

func numberColumn(fieldID, name string) ColumnSpec {
	return ColumnSpec{FieldID: fieldID, WidgetName: name, WidgetType: "number", Kind: KindNumber, Type: ColumnTypeNumeric}
}

func dateColumn(fieldID, name string) ColumnSpec {
	return ColumnSpec{FieldID: fieldID, WidgetName: name, WidgetType: "datetime", Kind: KindDate, Type: ColumnTypeDate}
}

func validParent() *StorageModel {
	return parentModel("tn_fd_01ab_k7q2m8",
		column("aaaaaaaa01", "_widget_a", KindText),
		numberColumn("aaaaaaaa02", "_widget_n"),
		dateColumn("aaaaaaaa03", "_widget_d"),
	)
}

// 首次发布：create_parent 动作携带全部列，合并模型即目标模型。
func TestDiffFirstPublishCreatesParentWithAllColumns(t *testing.T) {
	target := validParent()
	result, err := Diff(nil, target)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if result.TypeConflict != nil {
		t.Fatalf("unexpected type conflict: %v", result.TypeConflict)
	}
	if result.Plan.Empty {
		t.Fatal("first publish must not be empty")
	}
	if len(result.Plan.Actions) != 1 || result.Plan.Actions[0].Kind != ActionCreateParent {
		t.Fatalf("expect single create_parent action, got %+v", result.Plan.Actions)
	}
	if len(result.MergedModel.Columns) != 3 {
		t.Fatalf("merged model keeps all columns, got %d", len(result.MergedModel.Columns))
	}
	if len(result.AddedColumns) != 3 {
		t.Fatalf("added columns diagnostic, got %v", result.AddedColumns)
	}
}

// 新增字段：仅产出加列动作；用户列描述不带 NOT NULL（required 只是提交语义）。
func TestDiffAddedColumn(t *testing.T) {
	applied := validParent()
	target := validParent()
	target.Columns = append(target.Columns, column("aaaaaaaa04", "_widget_new", KindText))
	result, err := Diff(applied, target)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if len(result.Plan.Actions) != 1 || result.Plan.Actions[0].Kind != ActionAddParentColumn {
		t.Fatalf("expect single add_parent_column, got %+v", result.Plan.Actions)
	}
	if result.Plan.Actions[0].Column.WidgetName != "_widget_new" {
		t.Fatalf("added column mismatch: %+v", result.Plan.Actions[0].Column)
	}
}

// 删除字段：无 DDL 动作；弃用列保留在合并模型并标记 deprecated（物理结构
// 是历次已应用字段的并集，方案 §6）。
func TestDiffRemovedFieldKeepsDeprecatedColumn(t *testing.T) {
	applied := validParent()
	target := parentModel(applied.TableName,
		applied.Columns[0], applied.Columns[1], // 移除 _widget_d
	)
	result, err := Diff(applied, target)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if !result.Plan.Empty {
		t.Fatalf("field removal must not produce DDL, got %+v", result.Plan.Actions)
	}
	if len(result.MergedModel.Columns) != 3 {
		t.Fatalf("merged model must keep deprecated column, got %d", len(result.MergedModel.Columns))
	}
	deprecated := result.MergedModel.Columns[2]
	if !deprecated.Deprecated || deprecated.WidgetName != "_widget_d" {
		t.Fatalf("expected _widget_d deprecated, got %+v", deprecated)
	}
	if len(result.DeprecatedColumns) != 1 || result.DeprecatedColumns[0] != "_widget_d" {
		t.Fatalf("deprecated diagnostic mismatch: %v", result.DeprecatedColumns)
	}
}

// 无变化：空计划（同步发布路径），弃用列在两侧一致时同样空。
func TestDiffNoChangeIsEmptyPlan(t *testing.T) {
	applied := validParent()
	result, err := Diff(applied, validParent())
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if !result.Plan.Empty {
		t.Fatalf("identical models must yield empty plan, got %+v", result.Plan.Actions)
	}
	// 已弃用列再次 Diff：目标仍无该字段 → 维持弃用，不重复产出动作
	removed := parentModel(applied.TableName, applied.Columns[0], applied.Columns[1])
	first, err := Diff(applied, removed)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	again, err := Diff(&first.MergedModel, removed)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if !again.Plan.Empty {
		t.Fatalf("re-diff against deprecated column must stay empty, got %+v", again.Plan.Actions)
	}
}

// 字段类型改变：fieldId 相同而 kind/type 不同 → TypeConflict（不产出动作），
// 由调用方映射 FORM_STORAGE_TYPE_CHANGE_UNSUPPORTED（方案 §16.11）。
func TestDiffTypeChangeRejected(t *testing.T) {
	applied := validParent()
	target := validParent()
	target.Columns[1] = column("aaaaaaaa02", "_widget_n", KindText) // number → text
	result, err := Diff(applied, target)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if result.TypeConflict == nil {
		t.Fatal("type change must conflict")
	}
	if !strings.Contains(result.TypeConflict.FieldID, "aaaaaaaa02") {
		t.Fatalf("conflict field mismatch: %+v", result.TypeConflict)
	}
}

// datetime 形状变化（date → datetime）同样视为类型变化拒绝。
func TestDiffDatetimeShapeChangeRejected(t *testing.T) {
	applied := validParent()
	target := validParent()
	target.Columns[2] = ColumnSpec{FieldID: "aaaaaaaa03", WidgetName: "_widget_d", WidgetType: "datetime", Kind: KindDateTime, Type: ColumnTypeTimestamp}
	result, err := Diff(applied, target)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if result.TypeConflict == nil {
		t.Fatal("datetime shape change must conflict")
	}
}

// 表名不可变更：父表/子表名改变直接报错（表名永久不变，方案 §7）。
func TestDiffTableNameImmutable(t *testing.T) {
	applied := validParent()
	target := validParent()
	target.TableName = "tn_fd_01ab_zzz999"
	if _, err := Diff(applied, target); err == nil {
		t.Fatal("parent table rename must error")
	}
}

// 子表单：新增子表 → create_child；既有子表加列 → add_child_column；
// 子表消失 → 整体弃用；子表列类型变化 → 冲突。
func TestDiffChildTableLifecycle(t *testing.T) {
	childColumns := []ColumnSpec{
		column("bbbbbbbb01", "_widget_c1", KindText),
		numberColumn("bbbbbbbb02", "_widget_c2"),
	}
	target := validParent()
	target.Children = []ChildTableSpec{{
		FieldID: "cccccccc01", WidgetName: "_widget_sub", TableName: "tn_fc_01ab_p3n9v2",
		Columns: childColumns,
	}}

	// 首次：create_child
	first, err := Diff(nil, target)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if len(first.Plan.Actions) != 2 || first.Plan.Actions[1].Kind != ActionCreateChild {
		t.Fatalf("expect create_parent + create_child, got %+v", first.Plan.Actions)
	}

	// 既有子表加列：仅 add_child_column
	extended := validParent()
	extended.Children = []ChildTableSpec{{
		FieldID: "cccccccc01", WidgetName: "_widget_sub", TableName: "tn_fc_01ab_p3n9v2",
		Columns: append(append([]ColumnSpec{}, childColumns...), column("bbbbbbbb03", "_widget_c3", KindText)),
	}}
	added, err := Diff(&first.MergedModel, extended)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if len(added.Plan.Actions) != 1 || added.Plan.Actions[0].Kind != ActionAddChildColumn {
		t.Fatalf("expect add_child_column only, got %+v", added.Plan.Actions)
	}

	// 子表删除：整体弃用（无 DDL，物理表保留）
	dropped := validParent()
	droppedResult, err := Diff(&first.MergedModel, dropped)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if !droppedResult.Plan.Empty {
		t.Fatalf("child drop must not produce DDL, got %+v", droppedResult.Plan.Actions)
	}
	if len(droppedResult.MergedModel.Children) != 1 || !droppedResult.MergedModel.Children[0].Deprecated {
		t.Fatalf("dropped child must stay deprecated in merged model")
	}

	// 子表列类型变化：冲突
	retyped := validParent()
	retyped.Children = []ChildTableSpec{{
		FieldID: "cccccccc01", WidgetName: "_widget_sub", TableName: "tn_fc_01ab_p3n9v2",
		Columns: []ColumnSpec{childColumns[0], column("bbbbbbbb02", "_widget_c2", KindText)},
	}}
	conflicted, err := Diff(&first.MergedModel, retyped)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if conflicted.TypeConflict == nil {
		t.Fatal("child column type change must conflict")
	}

	// 子表名不可变更
	renamed := validParent()
	renamed.Children = []ChildTableSpec{{
		FieldID: "cccccccc01", WidgetName: "_widget_sub", TableName: "tn_fc_01ab_zzz999",
		Columns: childColumns,
	}}
	if _, err := Diff(&first.MergedModel, renamed); err == nil {
		t.Fatal("child table rename must error")
	}
}

// checksum 确定性：字段顺序无关、相同模型同值；模型变更后改变。
func TestModelChecksumDeterministic(t *testing.T) {
	forward := validParent()
	reversed := parentModel("tn_fd_01ab_k7q2m8",
		forward.Columns[2], forward.Columns[1], forward.Columns[0])
	a, err := ModelChecksum(forward)
	if err != nil {
		t.Fatalf("checksum: %v", err)
	}
	b, err := ModelChecksum(reversed)
	if err != nil {
		t.Fatalf("checksum: %v", err)
	}
	if a != b {
		t.Fatal("checksum must be order-independent")
	}
	if len(a) != 64 {
		t.Fatalf("checksum is sha256 hex, got %d chars", len(a))
	}
	changed := validParent()
	changed.Columns[0].Deprecated = true
	c, err := ModelChecksum(changed)
	if err != nil {
		t.Fatalf("checksum: %v", err)
	}
	if a == c {
		t.Fatal("checksum must change with model content")
	}
}

// Validate 非法输入：标识符白名单、类型/语义一致、fieldId 与 widgetName 去重、
// 父子表名前缀（方案 §16.7 的模型侧防线）。
func TestValidateRejectsInvalidModels(t *testing.T) {
	cases := []struct {
		name  string
		model *StorageModel
		want  string
	}{
		{"父表名前缀错误", parentModel("tn_fc_01ab_k7q2m8", column("aaaaaaaa01", "_widget_a", KindText)), "tn_fd_"},
		{"父表名随机段非法", parentModel("tn_fd_01ab_K7Q2M8", column("aaaaaaaa01", "_widget_a", KindText)), "规则"},
		{"fieldId 长度非法", parentModel("tn_fd_01ab_k7q2m8", column("short0001", "_widget_a", KindText)), "fieldId"},
		{"列名与 fieldId 不一致", &StorageModel{TableName: "tn_fd_01ab_k7q2m8", Columns: []ColumnSpec{{FieldID: "aaaaaaaa01", WidgetName: "_widget_a", WidgetType: "text", Kind: KindText, Type: ColumnTypeNumeric}}}, "不一致"},
		{"fieldId 重复", &StorageModel{TableName: "tn_fd_01ab_k7q2m8", Columns: []ColumnSpec{
			column("aaaaaaaa01", "_widget_a", KindText),
			column("aaaaaaaa01", "_widget_b", KindText),
		}}, "重复"},
		{"widgetName 重复", &StorageModel{TableName: "tn_fd_01ab_k7q2m8", Columns: []ColumnSpec{
			column("aaaaaaaa01", "_widget_a", KindText),
			column("aaaaaaaa02", "_widget_a", KindText),
		}}, "重复"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.model.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error %q must contain %q", err.Error(), testCase.want)
			}
		})
	}
}

// 标识符白名单：动态表名/列名/索引名的合法与非法形态。
func TestIdentifierWhitelist(t *testing.T) {
	if err := ValidateDynamicTableName("tn_fd_01ab_k7q2m8"); err != nil {
		t.Fatalf("valid parent table: %v", err)
	}
	if err := ValidateDynamicTableName("tn_fc_01ab_p3n9v2"); err != nil {
		t.Fatalf("valid child table: %v", err)
	}
	for _, name := range []string{"tn_fd_01AB_k7q2m8", "tn_fx_01ab_k7q2m8", "tn_fd_1ab_k7q2m8", "public.tn_fd_01ab_k7q2m8", "tn_fd_01ab_k7q2m8;--"} {
		if err := ValidateDynamicTableName(name); err == nil {
			t.Fatalf("%q must be rejected", name)
		}
	}
	if err := ValidateColumnName("f_aaaaaaaa01"); err != nil {
		t.Fatalf("valid column: %v", err)
	}
	for _, name := range []string{"f_Short", "f_aaaaaaaa0", "values", "f_aaaaaaaa01; DROP TABLE x"} {
		if err := ValidateColumnName(name); err == nil {
			t.Fatalf("%q must be rejected", name)
		}
	}
	if err := ValidateFieldID("aaaaaaaa01"); err != nil {
		t.Fatalf("valid fieldId: %v", err)
	}
	if err := ValidateFieldID("AAAAAAAA01"); err == nil {
		t.Fatal("uppercase fieldId must be rejected")
	}
}
