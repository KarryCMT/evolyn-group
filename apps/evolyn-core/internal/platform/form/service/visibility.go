// 有效可见性合成（v6 设计方案 §4.2）：承接 v5 显隐规则求值器，输出
//
//	effectiveVisible(F) = schemaVisible(F) ∧ permissionVisible(F, operation)
//	                     ∧ showRuleVisible(F, submittedValues, trustedContext)
//
// 字段权限是硬上限（权限隐藏的条件源条件不成立，下游不得反推其值）；
// 反之无权限查看的字段仍由服务端执行其不可见策略——保证流程后续节点不会
// 因字段裁剪而意外清掉前序值。纯函数、不依赖 Gin/GORM。
package service

// effectiveFieldVisibility 计算全部顶层字段的有效可见性（§4.2）。
//
// fields 为发布快照字段视图（静态 visible 事实源）；permissionVisible 为
// 权限矩阵投影（nil 表示无权限组基线，全量放行）；valueOf 返回求值用字段
// 值（信封路径取提交 data，受信合并路径取合并值）；currentMemberID 为空
// 表示匿名/未注入（includeCurrentMember 不参与比较集合）。
func effectiveFieldVisibility(
	fields map[string]snapshotField,
	content map[string]any,
	permissionVisible func(name string) bool,
	valueOf func(name string) any,
	currentMemberID string,
) map[string]bool {
	if permissionVisible == nil {
		permissionVisible = func(string) bool { return true }
	}
	// 条件源基线 = 静态可见 ∧ 权限可见（权限永远高于显隐规则）。
	baseReadable := func(name string) bool {
		field, ok := fields[name]
		if !ok || !field.visible {
			return false
		}
		return permissionVisible(name)
	}
	ruleVisible := compileFieldShowRules(content).ruleFieldVisibility(baseReadable, valueOf, currentMemberID)

	visibility := make(map[string]bool, len(fields))
	for name, field := range fields {
		visible := field.visible && permissionVisible(name)
		if computed, ok := ruleVisible[name]; ok {
			visible = visible && computed
		}
		visibility[name] = visible
	}
	return visibility
}
