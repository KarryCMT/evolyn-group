// Package ipregion 提供离线 IP 归属地解析（登录日志「登录地」数据源）。
// 查询实现与离线数据来自 ip2region（github.com/lionsoul2014/ip2region，
// Apache-2.0），IPv4 库以 xdb 文件随仓库内嵌、整库常驻内存，解析零外部依赖；
// IPv6 库体积过大暂不内嵌，IPv6 地址解析回退「未知」（接口已预留，后续可加库）。
package ipregion

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// v4XDB 内嵌的 IPv4 离线库（来源 ip2region data/ip2region_v4.xdb）
//
//go:embed data/ip2region_v4.xdb
var v4XDB []byte

const (
	// UnknownLocation 无法解析（IPv6/非法输入/未收录段）时的统一展示文案
	UnknownLocation = "未知"
	// IntranetAddress 保留段（Reserved，回环/私网地址）的展示文案
	IntranetAddress = "内网地址"
)

// Resolver IP 归属地解析器：输入请求来源 IP，返回「广东省 深圳市」样式的
// 展示文案，失败回退 UnknownLocation。实现必须并发安全
type Resolver interface {
	Resolve(ip string) string
}

type resolver struct {
	mu sync.Mutex
	// xdb 官方实现注释明确「not thread safe」（ioCount 等字段共享可变），
	// 登录写入频率低，单实例互斥串行化足够，无需 searcher 池
	searcher *xdb.Searcher
}

// New 构造默认解析器：内嵌 v4 库全量加载（NewWithBuffer 纯内存查询）。
// 数据损坏时返回错误，由装配层 fail-fast 拒绝启动
func New() (Resolver, error) {
	searcher, err := xdb.NewWithBuffer(xdb.IPv4, v4XDB)
	if err != nil {
		return nil, err
	}
	return &resolver{searcher: searcher}, nil
}

// Resolve 解析 IP 归属地；任何失败静默回退「未知」，不向调用方报错
// （登录日志的归属地列是尽力而为的增强信息，不构成写路径的失败条件）
func (r *resolver) Resolve(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return UnknownLocation
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	region, err := r.searcher.Search(ip)
	if err != nil {
		return UnknownLocation
	}
	return FormatRegion(region)
}

// FormatRegion 把 xdb v3 管道串（国家|省/州|市|ISP|国家代码）规整为展示文案：
// 优先取省/市两段（跳过占位 0、相邻同名去重，如直辖市「北京市|北京市」→「北京市」）；
// 省市全缺时回退国家段（境外单级标注常见）；保留段（Reserved，回环/私网）
// 映射为「内网地址」；均不可用时回退「未知」
func FormatRegion(region string) string {
	parts := strings.Split(region, "|")
	pick := func(i int) string {
		if i >= len(parts) {
			return ""
		}
		p := strings.TrimSpace(parts[i])
		if p == "" || p == "0" {
			return ""
		}
		return p
	}

	if country := pick(0); country == "Reserved" {
		return IntranetAddress
	}

	names := make([]string, 0, 2)
	for _, name := range []string{pick(1), pick(2)} {
		if name == "" {
			continue
		}
		if len(names) > 0 && names[len(names)-1] == name {
			continue
		}
		names = append(names, name)
	}
	if len(names) > 0 {
		return strings.Join(names, " ")
	}
	if country := pick(0); country != "" {
		return country
	}
	return UnknownLocation
}
