package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultQuotas(t *testing.T) {
	// free 兜底：未知套餐按 free
	free := DefaultQuotas("unknown")
	assert.Equal(t, int64(3), free[QuotaApps])

	trial := DefaultQuotas(PlanTrial)
	assert.Equal(t, int64(30), trial[QuotaMembers]) // 对齐简道云试用版

	pro := DefaultQuotas(PlanPro)
	assert.Equal(t, int64(-1), pro[QuotaStorageGB]) // pro 不限量
}

func TestQuotasGet(t *testing.T) {
	q := Quotas{QuotaMembers: 8}

	// 覆盖值优先
	assert.Equal(t, int64(8), q.Get(PlanTrial, QuotaMembers, -2))
	// 缺键回落套餐默认
	assert.Equal(t, int64(50), q.Get(PlanTrial, QuotaForms, -2))
	// 套餐默认也缺的键回落 def
	assert.Equal(t, int64(-2), q.Get(PlanTrial, "some_future_key", -2))
}

func TestQuotasScanValue(t *testing.T) {
	// nil map 序列化为空对象，Scan(nil) 归一为空 map
	var q Quotas
	v, err := q.Value()
	assert.NoError(t, err)
	assert.Equal(t, []byte("{}"), v)

	assert.NoError(t, q.Scan(nil))
	assert.Equal(t, Quotas{}, q)

	assert.NoError(t, q.Scan([]byte(`{"apps":10}`)))
	assert.Equal(t, int64(10), q[QuotaApps])
}

func TestTenantConfigScan(t *testing.T) {
	var c TenantConfig
	assert.NoError(t, c.Scan(nil)) // NULL 列回落默认配置
	assert.Equal(t, false, c.Watermark.Enabled)
	assert.Equal(t, "dark", c.Theme.AppNaviColor)

	assert.NoError(t, c.Scan([]byte(`{"watermark":{"enabled":true,"color":"dark"},"locale":"en_us"}`)))
	assert.True(t, c.Watermark.Enabled)
	assert.Equal(t, "en_us", c.Locale)

	// Value/Scan roundtrip
	out, err := c.Value()
	assert.NoError(t, err)
	var c2 TenantConfig
	assert.NoError(t, c2.Scan(out))
	assert.Equal(t, c, c2)

	// 非 JSON 类型报错
	assert.Error(t, c2.Scan(123))
}
