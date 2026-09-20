package numeric

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var regexpDecimalDTO = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

// DecimalString 是 HTTP DTO 中业务小数的唯一承载类型。JSON 数字会在解析
// 时变成 float64 并损失精度，因此金额、税率与数量必须以 JSON string 传输。
//
// 该类型只接收十进制文本（不接受指数、空白或多余的正号），反序列化后会收敛
// 为 Numeric 的 canonical 表示；调用方不应将它替换成 float64。
type DecimalString string

// ParseDecimalString 把外部十进制文本规范化为 DTO 值。空字符串、科学计数法
// 和非字符串协议文本均不属于 API decimal-string 协议。
func ParseDecimalString(input string) (DecimalString, error) {
	if input == "" || strings.TrimSpace(input) != input ||
		strings.ContainsAny(input, "eE+") || !regexpDecimalDTO.MatchString(input) {
		return "", newErr(CodeInvalidNumber, "invalid decimal-string DTO: "+input)
	}
	n, err := Parse(input)
	if err != nil {
		return "", err
	}
	text, err := n.Serialize()
	if err != nil {
		return "", err
	}
	return DecimalString(text), nil
}

// Numeric 将 DTO 值转换为数值域对象；空 DTO 不是 null，属于调用方缺失字段。
func (d DecimalString) Numeric() (Numeric, error) {
	if d == "" {
		return Numeric{}, newErr(CodeInvalidNumber, "decimal-string DTO is empty")
	}
	return Parse(string(d))
}

func (d DecimalString) String() string { return string(d) }

// MarshalJSON 始终输出字符串，避免 Go 编码器将业务小数降级为 JSON number。
func (d DecimalString) MarshalJSON() ([]byte, error) {
	if _, err := d.Numeric(); err != nil {
		return nil, fmt.Errorf("marshal decimal-string DTO: %w", err)
	}
	return json.Marshal(string(d))
}

// UnmarshalJSON 拒绝 JSON number/null；只允许十进制字符串输入并完成 canonical
// 归一化，使 DTO 进入业务层后不再携带多余尾零或前导零。
func (d *DecimalString) UnmarshalJSON(raw []byte) error {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return fmt.Errorf("decimal-string DTO must be a JSON string: %w", err)
	}
	parsed, err := ParseDecimalString(text)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
