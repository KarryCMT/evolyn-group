package expression

import (
	"fmt"
	"strings"
)

// 模板插值协议（Phase 7 service 节点定版）：{{expr}} 形态的表达式段嵌入
// 字面文本，发布期经 Engine 预编译（Compile 产物随 CompiledDefinition 冻结），
// 运行期仅以白名单环境求值（form.*/starter.*/variables.*），禁止运行期重编译。
// 转义：\{\{ 输出字面 {{（不解释表达式）。

// TemplateSegment 模板段：Literal 与 Program 互斥。
type TemplateSegment struct {
	// Literal 字面文本（Program 非 nil 时忽略）
	Literal string
	// Program 预编译表达式（nil=字面段）
	Program Program
}

// ParseTemplate 解析并预编译插值模板：任意段表达式编译失败即返回
// ErrExpressionInvalid（发布校验拒绝）。空串返回单字面空段。
func ParseTemplate(engine Engine, source string) ([]TemplateSegment, error) {
	if engine == nil {
		engine = NewExprEngine()
	}
	segments := make([]TemplateSegment, 0, 2)
	var literal strings.Builder
	for i := 0; i < len(source); {
		// 转义段：\{\{ → 字面 {{
		if strings.HasPrefix(source[i:], `\{{`) {
			literal.WriteString(`{{`)
			i += 3
			continue
		}
		if strings.HasPrefix(source[i:], "{{") {
			end := strings.Index(source[i+2:], "}}")
			if end < 0 {
				return nil, &ErrExpressionInvalid{Cause: fmt.Errorf("模板 {{ 未闭合: %q", source)}
			}
			source := strings.TrimSpace(source[i+2 : i+2+end])
			if source == "" {
				return nil, &ErrExpressionInvalid{Cause: fmt.Errorf("模板表达式为空: %q", source)}
			}
			program, err := engine.Compile(source)
			if err != nil {
				return nil, err
			}
			segments = append(segments, TemplateSegment{Literal: literal.String()})
			literal.Reset()
			segments = append(segments, TemplateSegment{Program: program})
			i = i + 2 + end + 2
			continue
		}
		literal.WriteByte(source[i])
		i++
	}
	segments = append(segments, TemplateSegment{Literal: literal.String()})
	return segments, nil
}

// RenderTemplate 以白名单环境渲染模板：表达式结果按 %v 文本化（标量语义；
// 复杂结构渲染为其 Go 文本形态，业务上应仅对标量插值）。
func RenderTemplate(segments []TemplateSegment, env map[string]any) (string, error) {
	var builder strings.Builder
	for i := range segments {
		seg := &segments[i]
		if seg.Program == nil {
			builder.WriteString(seg.Literal)
			continue
		}
		value, err := seg.Program.Eval(env)
		if err != nil {
			return "", err
		}
		builder.WriteString(fmt.Sprintf("%v", value))
	}
	return builder.String(), nil
}

// HasTemplateExpression 模板是否含表达式段（校验器/安全检查使用）。
func HasTemplateExpression(source string) bool {
	return strings.Contains(source, "{{")
}
