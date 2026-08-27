package service

import (
	"strings"

	kernel "evolyn/internal/model"
	"evolyn/internal/platform/enterpriselog/model"
)

// csvBOM UTF-8 BOM：Excel 直接双击打开 CSV 时不乱码
const csvBOM = "\uFEFF"

// csvRowSeparator RFC 4180 行分隔
const csvRowSeparator = "\r\n"

// loginCSVHeader 登录日志导出列（与页面列表列一致）
var loginCSVHeader = []string{"登录人", "登录时间", "登录地", "登录平台", "IP"}

// operationCSVHeader 操作日志导出列
var operationCSVHeader = []string{"操作人", "操作时间", "日志范围", "操作类型", "操作详情", "IP"}

// clientLabel 登录平台展示文案（与前端映射同口径）
func clientLabel(client string) string {
	switch client {
	case model.ClientWeb:
		return "电脑网页版"
	case model.ClientWap:
		return "手机网页版"
	default:
		return "未知"
	}
}

// csvEscape RFC 4180 字段转义：含逗号/引号/换行的字段以引号包裹，引号翻倍
func csvEscape(field string) string {
	if strings.ContainsAny(field, ",\"\r\n") {
		return `"` + strings.ReplaceAll(field, `"`, `""`) + `"`
	}
	return field
}

// buildCSV 组装完整 CSV 文档：BOM + 表头 + 数据行（CRLF 分隔）
func buildCSV(header []string, rows [][]string) string {
	var b strings.Builder
	b.WriteString(csvBOM)
	b.WriteString(csvLine(header))
	for _, row := range rows {
		b.WriteString(csvLine(row))
	}
	return b.String()
}

func csvLine(fields []string) string {
	escaped := make([]string, len(fields))
	for i, f := range fields {
		escaped[i] = csvEscape(f)
	}
	return strings.Join(escaped, ",") + csvRowSeparator
}

// loginCSVRows 登录日志行 → CSV 字段（时间按 JSONTime 秒级东八区出网口径）
func loginCSVRows(rows []model.LoginLogRow) [][]string {
	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, []string{
			actorDisplayName(row.ActorNameSnapshot, row.DisplayName),
			kernel.JSONTime(row.CreatedAt).String(),
			row.Location,
			clientLabel(row.Client),
			row.IP,
		})
	}
	return out
}

// operationCSVRows 操作日志行 → CSV 字段（历史行按降级文案展示）
func operationCSVRows(rows []model.AuditLogRow) [][]string {
	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, []string{
			actorDisplayName(row.ActorNameSnapshot, row.DisplayName),
			kernel.JSONTime(row.CreatedAt).String(),
			categoryDisplay(row.CategoryCode),
			eventDisplay(row.EventCode),
			summaryDisplay(row.Summary),
			row.IP,
		})
	}
	return out
}

// actorDisplayName 展示快照优先，存量历史行回查当前昵称兜底
func actorDisplayName(snapshot, fallback string) string {
	if snapshot != "" {
		return snapshot
	}
	return fallback
}
