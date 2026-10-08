<script setup lang="ts">
import type { WorkflowTaskScope } from '~/types/app';
import { showToast, Empty as VanEmpty, Icon as VanIcon, Loading as VanLoading } from 'vant';
import { computed, ref, watch } from 'vue';
import { listStartedWorkflowInstances, listWorkflowTasks } from '~/api/workflow';

type CenterScope = WorkflowTaskScope | 'started-by-me';

const props = withDefaults(defineProps<{ demo?: boolean }>(), {
  demo: false,
});

interface CenterItem {
  id: number;
  number: string;
  title: string;
  meta: string;
  status: string;
  createdAt: string;
}

const scope = ref<CenterScope>('pending');
const loading = ref(false);
const errorMessage = ref('');
const items = ref<CenterItem[]>([]);

const scopeItems: Array<{ key: CenterScope; label: string }> = [
  { key: 'pending', label: '我的待办' },
  { key: 'completed', label: '我处理的' },
  { key: 'cc-to-me', label: '抄送我的' },
  { key: 'started-by-me', label: '我发起的' },
];

const emptyDescription = computed(
  () => `${scopeItems.find((item) => item.key === scope.value)?.label ?? '流程'}暂无数据`,
);

async function load(): Promise<void> {
  loading.value = true;
  errorMessage.value = '';
  if (props.demo) {
    // 本地示例应用没有后端流程实例，直接展示对应范围的空状态。
    items.value = [];
    loading.value = false;
    return;
  }
  try {
    if (scope.value === 'started-by-me') {
      const page = await listStartedWorkflowInstances();
      items.value = page.items.map((item) => ({
        id: item.id,
        number: item.instanceNo,
        title: '我发起的流程',
        meta: '由我发起',
        status: item.status,
        createdAt: item.createdAt,
      }));
    } else {
      const page = await listWorkflowTasks(scope.value);
      items.value = page.items.map((item) => ({
        id: item.id,
        number: item.instanceNo,
        title: item.title || item.nodeName,
        meta: `${item.starterName} · ${item.nodeName}`,
        status: item.status,
        createdAt: item.createdAt,
      }));
    }
  } catch (error) {
    items.value = [];
    errorMessage.value = error instanceof Error ? error.message : '流程列表加载失败';
  } finally {
    loading.value = false;
  }
}

watch([scope, () => props.demo], () => void load(), { immediate: true });
</script>

<template>
  <section class="workflow-center">
    <div class="workflow-scopes" role="tablist" aria-label="流程范围">
      <button
        v-for="item in scopeItems"
        :key="item.key"
        type="button"
        role="tab"
        :aria-selected="scope === item.key"
        :class="{ 'is-active': scope === item.key }"
        @click="scope = item.key"
      >
        {{ item.label }}
      </button>
    </div>

    <div v-if="loading" class="workflow-state">
      <VanLoading color="var(--van-primary-color)" />
    </div>
    <div v-else-if="errorMessage" class="workflow-state workflow-state--error">
      <p>{{ errorMessage }}</p>
      <button type="button" @click="load">
        重新加载
      </button>
    </div>
    <VanEmpty v-else-if="items.length === 0" image="search" :description="emptyDescription" />
    <div v-else class="workflow-list">
      <button v-for="item in items" :key="`${scope}-${item.id}`" type="button" @click="showToast('流程详情移动端正在建设中')">
        <span class="workflow-list__icon"><VanIcon name="records-o" /></span>
        <span class="workflow-list__body">
          <strong>{{ item.title }}</strong>
          <small>{{ item.number }} · {{ item.meta }}</small>
          <small>{{ item.createdAt }}</small>
        </span>
        <span class="workflow-list__status">{{ item.status }}</span>
        <VanIcon name="arrow" />
      </button>
    </div>
  </section>
</template>

<style scoped>
.workflow-center {
  min-height: 0;
  padding: 16px;
  overflow-y: auto;
  flex: 1;
}

.workflow-scopes {
  display: flex;
  padding: 4px;
  overflow-x: auto;
  background: #f1f3f6;
  border-radius: 22px;
}

.workflow-scopes button {
  min-width: 82px;
  height: 38px;
  padding: 0 14px;
  font-size: 14px;
  color: #667080;
  white-space: nowrap;
  background: transparent;
  border: 0;
  border-radius: 19px;
  flex: 1;
}

.workflow-scopes button.is-active {
  font-weight: 600;
  color: var(--van-primary-color);
  background: #fff;
  box-shadow: 0 2px 8px rgb(31 42 61 / 8%);
}

.workflow-state {
  display: grid;
  min-height: 300px;
  place-items: center;
}

.workflow-state--error {
  align-content: center;
  gap: 12px;
  color: #7a8492;
}

.workflow-state--error button {
  padding: 8px 18px;
  color: var(--van-primary-color);
  background: transparent;
  border: 1px solid var(--van-primary-color);
  border-radius: 18px;
}

.workflow-list {
  display: grid;
  padding-top: 14px;
  gap: 12px;
}

.workflow-list > button {
  display: flex;
  min-width: 0;
  padding: 16px;
  color: #1f2a3d;
  text-align: left;
  background: #fff;
  border: 1px solid #edf0f3;
  border-radius: 12px;
  box-shadow: 0 4px 16px rgb(31 42 61 / 5%);
  align-items: center;
}

.workflow-list__icon {
  display: grid;
  width: 42px;
  height: 42px;
  margin-right: 12px;
  font-size: 22px;
  color: var(--van-primary-color);
  background: color-mix(in srgb, var(--van-primary-color) 10%, white);
  border-radius: 9px;
  place-items: center;
}

.workflow-list__body {
  display: grid;
  min-width: 0;
  gap: 4px;
  flex: 1;
}

.workflow-list__body strong,
.workflow-list__body small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workflow-list__body strong {
  font-size: 16px;
}

.workflow-list__body small {
  font-size: 12px;
  color: #8b93a0;
}

.workflow-list__status {
  margin: 0 8px;
  font-size: 12px;
  color: var(--van-primary-color);
}
</style>
