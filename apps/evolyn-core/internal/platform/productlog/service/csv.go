package service

import (
	"strings"

	kernel "evolyn/internal/model"
	"evolyn/internal/platform/productlog/model"
)

// csvBOM UTF-8 BOM：Excel 直接双击打开 CSV 时不乱码
const csvBOM = "\uFEFF"

// csvRowSeparator RFC 4180 行分隔
const csvRowSeparator = "\r\n"

// productCSVHeader 产品日志导出列（与页面列表列一致）
var productCSVHeader = []string{"操作人", "操作时间", "日志范围", "操作类型", "所属应用", "操作对象", "操作详情", "IP"}

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

// productCSVRows 产品日志行 → CSV 字段（时间按 JSONTime 秒级东八区出网口径；
// 历史行按降级文案展示；非应用内操作的所属应用渲染「—」）
func productCSVRows(rows []model.ProductLogRow) [][]string {
	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		application := row.ApplicationNameSnapshot
		if application == "" {
			application = "—"
		}
		target := row.TargetNameSnapshot
		if target == "" {
			target = "—"
		}
		out = append(out, []string{
			actorDisplayName(row.ActorNameSnapshot, row.DisplayName),
			kernel.JSONTime(row.CreatedAt).String(),
			categoryDisplay(row.CategoryCode),
			eventDisplay(row.EventCode),
			application,
			target,
			summaryDisplay(row.Summary),
			row.IP,
		})
	}
	return out
}
