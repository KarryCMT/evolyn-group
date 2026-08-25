// Package totp RFC 6238 时间型一次性密码（TOTP）与密钥加密：
// 零第三方依赖（stdlib 实现），供账号安全域的 MFA 因子校验使用。
// 参数口径：HMAC-SHA1 / 6 位 / 30 秒步长，校验允许 ±1 窗口时钟偏移
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
)

const (
	Step  = 30 // 时间步长（秒）
	Digit = 6  // 码长
	skew  = 1  // 校验窗口偏移（±1 步）
)

var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateSecret 生成 160 位随机密钥并返回 base32 编码（兼容主流验证器）
func GenerateSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}
	return encoding.EncodeToString(buf), nil
}

// OtpauthURL 验证器扫码导入地址（otpauth:// 协议，Authy/Google
// Authenticator 等均支持）
func OtpauthURL(issuer, account, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		url.PathEscape(issuer), url.PathEscape(account), secret, url.PathEscape(issuer), Digit, Step)
}

// Code 按时间步生成 TOTP 码（测试与首码校验用）
func Code(secret string, t unix) (string, error) {
	key, err := encoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", fmt.Errorf("decode secret: %w", err)
	}

	counter := uint64(t / Step)
	mac := hmac.New(sha1.New, key)
	_ = binary.Write(mac, binary.BigEndian, counter)
	sum := mac.Sum(nil)

	// 动态截断（RFC 4226 §5.3）
	offset := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", code%1_000_000), nil
}

// unix 时间秒（别名收敛类型，防误传毫秒）
type unix int64

// Now 当前时间秒（包外只经 Verify 使用相对时间，测试可注入绝对时间）
func At(sec int64) unix { return unix(sec) }

// Verify 校验动态码：允许 ±1 窗口偏移；replay 保护——命中步长必须
// 大于 lastUsed（同一窗口的码不可重复使用）。返回命中的步长计数器
// （调用方持久化为 last_used_counter）
func Verify(secret, code string, at unix, lastUsed int64) (int64, bool, error) {
	if len(code) != Digit {
		return 0, false, nil
	}

	nowStep := int64(at) / Step
	for drift := -skew; drift <= skew; drift++ {
		counter := nowStep + int64(drift)
		if counter <= lastUsed {
			continue // 重放窗口：已用过的码不再接受
		}
		expect, err := Code(secret, unix(counter*Step))
		if err != nil {
			return 0, false, err
		}
		// 恒定时间比较防时序侧信道
		if subtle.ConstantTimeCompare([]byte(expect), []byte(code)) == 1 {
			return counter, true, nil
		}
	}
	return 0, false, nil
}
