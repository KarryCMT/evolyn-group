package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// 菜单收藏（个人状态动作，ADR-011/P1-P2）业务指标：收藏是频繁的个人
// 偏好操作，不写 tn_audit_logs（避免污染企业操作审计），以计数器观测
// 写入/取消/列表与可见性拒绝规模（菜单收藏方案 §7.3）。日志只记录租户、
// 成员与公开编码，不记录表单数据。
var (
	// MenuFavoriteCreateTotal 收藏写入成功次数（含幂等重复收藏）
	MenuFavoriteCreateTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "menu_favorite_create_total",
		Help: "Total number of menu favorite creations (idempotent repeats included).",
	})

	// MenuFavoriteRemoveTotal 取消收藏次数（含幂等重复取消）
	MenuFavoriteRemoveTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "menu_favorite_remove_total",
		Help: "Total number of menu favorite removals (idempotent repeats included).",
	})

	// MenuFavoriteListTotal 我的收藏列表读取次数
	MenuFavoriteListTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "menu_favorite_list_total",
		Help: "Total number of menu favorite list reads.",
	})

	// MenuFavoriteVisibilityRejectedTotal 收藏写入被可见性策略拒绝的次数
	//（隐藏节点/资产软删/表单入口权限失效/分组节点等 canFavorite 不满足）
	MenuFavoriteVisibilityRejectedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "menu_favorite_visibility_rejected_total",
		Help: "Total number of menu favorite creations rejected by visibility policy.",
	})
)

func init() {
	prometheus.Register(MenuFavoriteCreateTotal)
	prometheus.Register(MenuFavoriteRemoveTotal)
	prometheus.Register(MenuFavoriteListTotal)
	prometheus.Register(MenuFavoriteVisibilityRejectedTotal)
}
