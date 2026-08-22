package ipregion

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 离线库单测直接跑内嵌 xdb（无网络/外部依赖）；公网 IP 只断言格式不断言
// 具体城市——xdb 数据随版本更新可能漂移，仓内文件是唯一稳定事实
func TestResolve(t *testing.T) {
	r, err := New()
	assert.NoError(t, err)

	assert.Equal(t, UnknownLocation, r.Resolve(""))
	assert.Equal(t, UnknownLocation, r.Resolve("not-an-ip"))
	assert.Equal(t, UnknownLocation, r.Resolve("240e:978:eee:ee::1"), "IPv6 库未内嵌，应回退未知")

	// 回环/私网段在 xdb 中标注为 Reserved 保留段
	assert.Equal(t, IntranetAddress, r.Resolve("127.0.0.1"))
	assert.Equal(t, IntranetAddress, r.Resolve("192.168.1.100"))

	// 公网 IP 必须解析出非未知归属地，且文案不含管道分隔符
	for _, ip := range []string{"110.242.68.66", "210.21.226.222", "8.8.8.8"} {
		location := r.Resolve(ip)
		assert.NotEqual(t, UnknownLocation, location, ip)
		assert.NotContains(t, location, "|", ip)
		t.Logf("%s -> %s", ip, location)
	}
}

func TestFormatRegion(t *testing.T) {
	cases := []struct {
		name   string
		region string
		want   string
	}{
		{"省+市", "中国|广东省|深圳市|电信|CN", "广东省 深圳市"},
		{"直辖市去重", "中国|北京市|北京市|联通|CN", "北京市"},
		{"仅省级", "中国|浙江省|0|电信|CN", "浙江省"},
		{"境外州市", "United States|California|0|Google LLC|US", "California"},
		{"保留段", "Reserved|Reserved|Reserved|0|0", IntranetAddress},
		{"全占位", "0|0|0|0|0", UnknownLocation},
		{"段缺失容忍", "中国", "中国"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, FormatRegion(c.region))
		})
	}
}

// 并发安全冒烟：并发 Resolve 不应 panic/死锁（xdb 官方实现非线程安全，靠互斥锁保证）
func TestResolveConcurrent(t *testing.T) {
	r, err := New()
	assert.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Resolve("110.242.68.66")
		}()
	}
	wg.Wait()
}
