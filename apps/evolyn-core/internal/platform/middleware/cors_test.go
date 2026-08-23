package middleware

import "testing"

// TestBuildAllowOriginFunc CORS Origin 判定（P1 整改）：
// 白名单精确匹配；宽松模式只放行本机回环任意端口；未知来源一律拒绝
func TestBuildAllowOriginFunc(t *testing.T) {
	allow := buildAllowOriginFunc([]string{"https://www.example.com", "http://localhost:5173 "}, false)

	cases := []struct {
		origin string
		want   bool
	}{
		{"https://www.example.com", true},           // 白名单精确命中（含协议与端口）
		{"https://example.com", false},              // 裸域不等于子域
		{"http://localhost:5173", true},             // 配置项带空白会被规整后命中
		{"https://www.example.com.evil.com", false}, // 后缀拼贴攻击
		{"http://localhost:5174", false},            // 严格模式不放行白名单外的回环端口
		{"", false},
	}
	for _, c := range cases {
		if got := allow(c.origin); got != c.want {
			t.Errorf("allow(%q) = %v, want %v", c.origin, got, c.want)
		}
	}
}

// TestBuildAllowOriginFuncDevLoose 宽松模式（仅 debug 环境传入 true）：
// 本机回环任意端口放行，非回环地址仍拒绝
func TestBuildAllowOriginFuncDevLoose(t *testing.T) {
	allow := buildAllowOriginFunc(nil, true)

	for _, origin := range []string{
		"http://localhost:5173", "http://localhost:3000",
		"https://127.0.0.1:8080", "http://127.0.0.1",
	} {
		if !allow(origin) {
			t.Errorf("devLoose should allow loopback origin %q", origin)
		}
	}
	for _, origin := range []string{
		"http://192.168.1.10:5173", "https://evil.com", "http://localhost.evil.com",
	} {
		if allow(origin) {
			t.Errorf("devLoose must not allow non-loopback origin %q", origin)
		}
	}
}
