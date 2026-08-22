package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure/ipregion"
	kernel "evolyn/internal/model"
	"evolyn/internal/platform/auth/loginlog/model"
	"evolyn/internal/platform/auth/loginlog/repository"

	"github.com/sirupsen/logrus"
)

// 日期入参与出网同一口径：yyyy-MM-dd 按东八区解析为自然日零点
const dateLayout = "2006-01-02"

// 分页默认值与上限（与 API 文档一致）
const (
	defaultPageSize = 20
	maxPageSize     = 100
)

type loginLogService struct {
	repo     repository.LoginLogRepository
	resolver ipregion.Resolver
}

// NewService 登录日志域服务工厂（ADR-007 域模块化）：同一实例实现写入
// （Recorder）与账号自查查询（QueryService）两个契约
func NewService(repo repository.LoginLogRepository, resolver ipregion.Resolver) *loginLogService {
	return &loginLogService{repo: repo, resolver: resolver}
}

// Record 落一条登录日志：IP/UA/RequestID 从请求元数据（RequestInfoMiddleware
// 注入）自动补全，客户端形态由 UA 解析、归属地由 IP 写时离线解析。
// 写入失败仅记录错误日志——登录日志不可用不应阻断登录主流程
func (s *loginLogService) Record(ctx context.Context, e Entry) {
	meta := contextx.RequestMetaFromContext(ctx)

	entry := &model.LoginLog{
		AccountID: e.AccountID,
		TenantID:  e.TenantID,
		MemberID:  e.MemberID,
		Method:    e.Method,
		Client:    ParseClient(meta.UserAgent),
		IP:        meta.IP,
		Location:  s.resolver.Resolve(meta.IP),
		UserAgent: meta.UserAgent,
		RequestID: meta.RequestID,
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		logrus.Warnf("login log persist failed: accountId=%d method=%s: %v", e.AccountID, e.Method, err)
	}
}

// ListByAccount 账号维度登录日志分页查询：规范化分页参数，日期字符串按
// 东八区解析为自然日闭区间（EndDate 换算为次日零点开区间上界）后交仓储
func (s *loginLogService) ListByAccount(ctx context.Context, accountID uint, q PageQuery) (*PageResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = defaultPageSize
	}
	if q.PageSize > maxPageSize {
		q.PageSize = maxPageSize
	}

	var start, endExcl time.Time
	if q.StartDate != "" {
		parsed, err := time.ParseInLocation(dateLayout, q.StartDate, kernel.CSTLocation())
		if err != nil {
			return nil, fmt.Errorf("startDate 格式非法，应为 yyyy-MM-dd")
		}
		start = parsed
	}
	if q.EndDate != "" {
		parsed, err := time.ParseInLocation(dateLayout, q.EndDate, kernel.CSTLocation())
		if err != nil {
			return nil, fmt.Errorf("endDate 格式非法，应为 yyyy-MM-dd")
		}
		endExcl = parsed.AddDate(0, 0, 1)
	}

	items, total, err := s.repo.ListByAccount(ctx, accountID, start, endExcl, (q.Page-1)*q.PageSize, q.PageSize)
	if err != nil {
		return nil, err
	}
	return &PageResult{Items: items, Total: total}, nil
}

// ParseClient 从 User-Agent 粗解析客户端形态：含移动端特征词归 wap（平板
// 一并归入），其余非空 UA 归 web，空 UA 为 unknown。仅服务展示分类，
// 不做精确设备识别
func ParseClient(ua string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return model.ClientUnknown
	}
	ua = strings.ToLower(ua)
	for _, keyword := range []string{"mobile", "android", "iphone", "ipod", "ipad"} {
		if strings.Contains(ua, keyword) {
			return model.ClientWap
		}
	}
	return model.ClientWeb
}
