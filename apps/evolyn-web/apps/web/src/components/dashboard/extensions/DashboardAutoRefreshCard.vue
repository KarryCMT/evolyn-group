<script setup lang="ts">
import { shallowRef } from 'vue';

defineOptions({ name: 'DashboardAutoRefreshCard' });

const enabled = shallowRef(false);
const interval = shallowRef('15');

const intervalOptions = [
  { value: '5', label: '5分钟' },
  { value: '10', label: '10分钟' },
  { value: '15', label: '15分钟' },
  { value: '30', label: '30分钟' },
  { value: '60', label: '1小时' },
];
</script>

<template>
  <section class="dashboard-extension-card" aria-labelledby="auto-refresh-title">
    <header class="dashboard-extension-card__header">
      <h2 id="auto-refresh-title">
        自动刷新
      </h2>
      <p>在「全屏模式」下，按照一定时间间隔自动刷新仪表盘图表</p>
    </header>

    <div class="dashboard-extension-card__body">
      <el-switch v-model="enabled" aria-label="启用自动刷新" />
      <div v-if="enabled" class="dashboard-extension-card__settings">
        <label class="dashboard-extension-card__label" for="refresh-interval">刷新间隔</label>
        <el-select
          id="refresh-interval"
          v-model="interval"
          class="dashboard-extension-card__control"
          aria-label="刷新间隔"
        >
          <el-option
            v-for="option in intervalOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </el-select>
        <p class="dashboard-extension-card__hint">
          自动刷新仅在「全屏模式」下生效
        </p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.dashboard-extension-card {
  overflow: hidden;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 2px;
  box-shadow: 0 2px 8px rgb(31 45 61 / 6%);
}

.dashboard-extension-card__header {
  display: flex;
  min-height: 64px;
  padding: 0 28px;
  align-items: center;
  gap: 14px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.dashboard-extension-card__header h2,
.dashboard-extension-card__header p,
.dashboard-extension-card__hint {
  margin: 0;
}

.dashboard-extension-card__header h2 {
  flex: 0 0 auto;
  font-size: 16px;
  font-weight: 650;
  color: var(--el-text-color-primary);
}

.dashboard-extension-card__header p,
.dashboard-extension-card__hint {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.dashboard-extension-card__body {
  min-height: 84px;
  padding: 26px 28px;
}

.dashboard-extension-card__settings {
  width: min(380px, 100%);
  margin-top: 14px;
}

.dashboard-extension-card__label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}

.dashboard-extension-card__control {
  width: 100%;
}

.dashboard-extension-card__hint {
  margin-top: 12px;
}

@media (width <= 640px) {
  .dashboard-extension-card__header {
    padding: 14px 18px;
    align-items: flex-start;
    flex-direction: column;
    gap: 4px;
  }

  .dashboard-extension-card__body {
    padding: 22px 18px;
  }
}
</style>
