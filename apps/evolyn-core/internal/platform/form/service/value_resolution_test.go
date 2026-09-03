package service

import (
	"testing"

	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
)

// v6 不可见字段赋值值决议器用例（docs/低代码平台/表单设计器/不可见字段赋值
// 前后端设计方案.md §4/§7）：三种策略、伪造信封拒绝、权限矩阵交集与
// 流程受信合并路径。content 构造复用 value_test.go 的 snapshot/snapItem。

// v6Content 在快照上叠加 v6 策略键（信封/合并路径直接消费文档，不走结构校验）。
func v6Content(items ...map[string]any) map[string]any {
	content := snapshot(items...)
	root := content["content"].(map[string]any)
	root["fieldShowRules"] = []any{}
	root["submitRule"] = 2
	root["widget_submit_rules"] = map[string]any{}
	return content
}

func withSubmitRules(content map[string]any, submitRule int, special map[string]any) map[string]any {
	root := content["content"].(map[string]any)
	root["submitRule"] = submitRule
	root["widget_submit_rules"] = special
	return content
}

// §7.1 条件切换清空旧值：规则隐藏字段按默认「空值」写入类型化空值。
func TestResolveSubmittedValuesClearByDefault(t *testing.T) {
	content := v6Content(
		snapItem("radiogroup", "_widget_status", "任务状态", map[string]any{
			"options": []any{map[string]any{"label": "未开始", "value": "todo"}, map[string]any{"label": "进行中", "value": "doing"}},
		}),
		snapItem("text", "_widget_eta", "预计处理时间", nil),
		snapItem("checkboxgroup", "_widget_tags", "标签", map[string]any{
			"options": []any{map[string]any{"label": "A", "value": "a"}},
		}),
	)
	rules := []any{map[string]any{
		"id": "_field_show_rule_1",
		"filter": map[string]any{
			"rel": "and",
			"cond": []any{map[string]any{
				"field": "_widget_status", "type": "radiogroup", "method": "eq", "value": []any{"todo"},
			}},
		},
		"fields": []any{"_widget_eta", "_widget_tags"},
	}}
	content["content"].(map[string]any)["fieldShowRules"] = rules

	// 状态=todo：eta/tags 规则可见，正常提交。
	cleaned, errs := ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_status": wrappedValue(`"todo"`, true),
		"_widget_eta":    wrappedValue(`"周五前"`, true),
		"_widget_tags":   wrappedValue(`["a"]`, true),
	}, nil, nil, "")
	assert.Empty(t, errs)
	assert.Equal(t, "周五前", cleaned["_widget_eta"])
	assert.Equal(t, []any{"a"}, cleaned["_widget_tags"])

	// 状态=doing：eta/tags 隐藏，客户端不带 data；服务端按默认「空值」写入
	// 类型化空值（单值 null、多选 []），键保留以区分「存在但为空」。
	cleaned, errs = ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_status": wrappedValue(`"doing"`, true),
		"_widget_eta":    wrappedValue("", false),
		"_widget_tags":   wrappedValue("", false),
	}, nil, nil, "")
	assert.Empty(t, errs)
	assert.Nil(t, cleaned["_widget_eta"])
	assert.Equal(t, []any{}, cleaned["_widget_tags"])
}

// §7.2 流程节点保留前序字段：preserve 只取锁定基线，不信任客户端载荷。
// 审批人对两字段无查看权限（权限隐藏 → 有效不可见），信封 visible=false。
func TestResolveSubmittedValuesPreserveBaseline(t *testing.T) {
	content := withSubmitRules(v6Content(
		snapItem("text", "_widget_code", "产品编号", nil),
		snapItem("text", "_widget_price", "定价", nil),
	), 2, map[string]any{"_widget_code": 1, "_widget_price": 1})
	approverMatrix := map[string]FieldPermission{
		"_widget_code":  {Visible: false, Editable: false},
		"_widget_price": {Visible: false, Editable: false},
	}

	// 编辑/流程后续提交：字段不可见（信封 visible=false），服务端从基线保留。
	baseline := map[string]any{"_widget_code": "P-001", "_widget_price": "199"}
	cleaned, errs := ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_code":  wrappedValue("", false),
		"_widget_price": wrappedValue("", false),
	}, approverMatrix, baseline, "")
	assert.Empty(t, errs)
	assert.Equal(t, "P-001", cleaned["_widget_code"])
	assert.Equal(t, "199", cleaned["_widget_price"])

	// 新建提交无旧值：preserve 得到类型空值（§3.2 设计器提示同口径）。
	cleaned, errs = ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_code":  wrappedValue("", false),
		"_widget_price": wrappedValue("", false),
	}, approverMatrix, nil, "")
	assert.Empty(t, errs)
	assert.Nil(t, cleaned["_widget_code"])
	assert.Nil(t, cleaned["_widget_price"])

	// 伪造隐藏 data：无论策略为何一律拒绝（隐藏值不能成为 preserve/recompute 来源）。
	_, errs = ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_code":  wrappedValue(`"P-黑客"`, false),
		"_widget_price": wrappedValue("", false),
	}, approverMatrix, baseline, "")
	assert.Equal(t, []string{"隐藏字段不能提交值"}, errs["_widget_code"])
}

// 信封伪造与必填边界：visible 与服务端有效可见性不一致、未知键、隐藏必填字段。
func TestResolveSubmittedValuesEnvelopeAndRequired(t *testing.T) {
	content := v6Content(
		snapItem("text", "_widget_name", "姓名", map[string]any{"allowBlank": false}),
		snapItem("text", "_widget_secret", "内部字段", map[string]any{
			"visible": false, "allowBlank": false,
		}),
	)

	// 静态隐藏必填字段：不执行必填校验（策略不能被绕过，也不阻断提交），
	// 按默认策略写入类型空值并保留键。
	cleaned, errs := ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_name":   wrappedValue(`"张三"`, true),
		"_widget_secret": wrappedValue("", false),
	}, nil, nil, "")
	assert.Empty(t, errs)
	assert.Nil(t, cleaned["_widget_secret"])

	// 伪造信封（隐藏字段声明 visible=true）→ 字段级错误。
	_, errs = ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_name":   wrappedValue(`"张三"`, true),
		"_widget_secret": wrappedValue("", true),
	}, nil, nil, "")
	assert.Equal(t, []string{"字段可见状态与发布快照不一致"}, errs["_widget_secret"])

	// 缺少信封与未知键。
	_, errs = ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_name": {Data: model.JSONContent(`"张三"`)},
	}, nil, nil, "")
	assert.Equal(t, []string{"缺少字段可见状态"}, errs["_widget_name"])
	_, errs = ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_name":  wrappedValue(`"张三"`, true),
		"_widget_ghost": wrappedValue(`"x"`, true),
	}, nil, nil, "")
	assert.Equal(t, []string{"提交了表单中不存在的字段"}, errs["_widget_ghost"])

	// 可见必填字段缺值 → 必填错误（可见字段的必填校验在策略之后执行）。
	_, errs = ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_name":   wrappedValue("", true),
		"_widget_secret": wrappedValue("", false),
	}, nil, nil, "")
	assert.Equal(t, []string{"请输入姓名"}, errs["_widget_name"])
}

// §4.2 权限交集：权限隐藏字段按策略决议；可见但不可编辑字段沿用基线合并。
func TestResolveSubmittedValuesWithPermissions(t *testing.T) {
	content := withSubmitRules(v6Content(
		snapItem("text", "_widget_name", "姓名", nil),
		snapItem("text", "_widget_memo", "备注", nil),
		snapItem("text", "_widget_hidden", "隐藏项", nil),
	), 2, map[string]any{"_widget_hidden": 1})

	permissions := map[string]FieldPermission{
		"_widget_name":   {Visible: true, Editable: true},
		"_widget_memo":   {Visible: true, Editable: false},
		"_widget_hidden": {Visible: false, Editable: false},
	}
	previous := map[string]any{"_widget_memo": "旧备注", "_widget_hidden": "保留值"}

	cleaned, errs := ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_name":   wrappedValue(`"张三"`, true),
		"_widget_memo":   wrappedValue("", true),
		"_widget_hidden": wrappedValue("", false),
	}, permissions, previous, "")
	assert.Empty(t, errs)
	assert.Equal(t, "张三", cleaned["_widget_name"])
	// 可见但不可编辑：服务端从基线合并。
	assert.Equal(t, "旧备注", cleaned["_widget_memo"])
	// 权限隐藏：有效不可见 → preserve 保留基线值。
	assert.Equal(t, "保留值", cleaned["_widget_hidden"])

	// 可见但不可编辑字段携带 data → 越权拒绝。
	_, errs = ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_name":   wrappedValue(`"张三"`, true),
		"_widget_memo":   wrappedValue(`"越权修改"`, true),
		"_widget_hidden": wrappedValue("", false),
	}, permissions, previous, "")
	assert.Equal(t, []string{permissionDeniedFieldMessage}, errs["_widget_memo"])

	// 信封伪造：权限隐藏字段声明 visible=true → 与有效可见性不一致。
	_, errs = ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_name":   wrappedValue(`"张三"`, true),
		"_widget_memo":   wrappedValue("", true),
		"_widget_hidden": wrappedValue("", true),
	}, permissions, previous, "")
	assert.Equal(t, []string{"字段可见状态与发布快照不一致"}, errs["_widget_hidden"])
}

// 流程受信合并路径（WorkflowRecordStore）：以合并值重算可见性并执行同一策略。
func TestResolveMergedRecordValues(t *testing.T) {
	content := v6Content(
		snapItem("radiogroup", "_widget_status", "任务状态", map[string]any{
			"options": []any{map[string]any{"label": "未开始", "value": "todo"}, map[string]any{"label": "进行中", "value": "doing"}},
		}),
		snapItem("text", "_widget_eta", "预计处理时间", nil),
		snapItem("text", "_widget_owner", "负责人", nil),
	)
	content["content"].(map[string]any)["fieldShowRules"] = []any{map[string]any{
		"id": "_field_show_rule_1",
		"filter": map[string]any{
			"rel": "and",
			"cond": []any{map[string]any{
				"field": "_widget_status", "type": "radiogroup", "method": "eq", "value": []any{"todo"},
			}},
		},
		"fields": []any{"_widget_eta"},
	}}

	// 基线：状态 todo、eta 有值；审批人将状态改为 doing → eta 规则隐藏，
	// 合并终审按默认「空值」清空，不残留旧预计时间。
	baseline := map[string]any{"_widget_status": "todo", "_widget_eta": "周五前", "_widget_owner": "李四"}
	merged := map[string]any{"_widget_status": "doing", "_widget_eta": "周五前", "_widget_owner": "王五"}
	cleaned, errs := ResolveMergedRecordValues(content, merged, baseline)
	assert.Empty(t, errs)
	assert.Nil(t, cleaned["_widget_eta"])
	assert.Equal(t, "doing", cleaned["_widget_status"])
	assert.Equal(t, "王五", cleaned["_widget_owner"])

	// 状态改回 todo：eta 规则重新可见，合并值通过终审保留。
	merged["_widget_status"] = "todo"
	cleaned, errs = ResolveMergedRecordValues(content, merged, baseline)
	assert.Empty(t, errs)
	assert.Equal(t, "周五前", cleaned["_widget_eta"])
}

// v6 快照的 recompute 配置在发布校验即被拒绝；防御路径按字段错误 fail-closed。
func TestResolveSubmittedValuesRecomputeFailClosed(t *testing.T) {
	content := withSubmitRules(v6Content(
		snapItem("text", "_widget_name", "姓名", nil),
		snapItem("number", "_widget_score", "总分", map[string]any{"visible": false}),
	), 2, map[string]any{"_widget_score": 3})
	_, errs := ResolveSubmittedValues(content, map[string]model.SubmitFieldValue{
		"_widget_name":  wrappedValue(`"张三"`, true),
		"_widget_score": wrappedValue("", false),
	}, nil, nil, "")
	assert.Equal(t, []string{"该字段的重算能力尚未开放"}, errs["_widget_score"])
}
