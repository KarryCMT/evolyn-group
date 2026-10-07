package metrics

import "github.com/prometheus/client_golang/prometheus"

// 仪表盘查询指标只记录稳定结果分类和规模，不记录筛选值、字段值或业务行。
// 低基数标签避免设计器中的组件、仪表盘编码造成 Prometheus 时序膨胀。
var (
	DashboardQueryTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dashboard_query_total",
		Help: "Total number of dashboard widget queries by stable result class.",
	}, []string{"result"})

	DashboardQueryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dashboard_query_duration_seconds",
		Help:    "Dashboard widget query duration by stable result class.",
		Buckets: prometheus.DefBuckets,
	}, []string{"result"})

	DashboardQueryRows = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "dashboard_query_rows",
		Help:    "Number of rows returned by successful dashboard widget queries.",
		Buckets: []float64{0, 1, 10, 20, 50, 100, 500, 1000, 5000},
	})
)

func init() {
	prometheus.Register(DashboardQueryTotal)
	prometheus.Register(DashboardQueryDuration)
	prometheus.Register(DashboardQueryRows)
}
