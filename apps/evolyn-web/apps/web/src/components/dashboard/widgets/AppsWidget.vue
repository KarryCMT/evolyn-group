<script setup lang="ts">
import {
  RiAddFill,
  RiBookmark3Fill,
  RiBriefcase4Fill,
  RiCheckboxCircleFill,
  RiContactsBook3Fill,
  RiPieChart2Fill,
  RiSearchFill,
} from '@remixicon/vue';
import { DashboardWidgetFrame } from '@evolyn.do/dashboard';
import { ApiError, ERROR_CODES } from '@evolyn.do/utils';
import { ElMessage } from 'element-plus';
import { computed, markRaw, onMounted, ref, shallowRef, type Component } from 'vue';
import { createBlankApplication, listApplications } from '~/api/applications';
import CreateApplicationDialog from '~/components/application/create/CreateApplicationDialog.vue';
import type { BlankApplicationDraft } from '~/components/application/create/BlankApplicationDialog.vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import type { ApplicationIcon, ApplicationItem } from '~/types';

defineOptions({ name: 'AppsWidget' });
const props = withDefaults(
  defineProps<{
    widget: DashboardWidgetContent;
    editorMode?: boolean;
  }>(),
  { editorMode: false },
);

// 图标键 → Remix Fill 图标（键值与后端服务端枚举一致，不存组件名）
const iconByKey: Record<ApplicationIcon, Component> = {
  bookmark: markRaw(RiBookmark3Fill),
  briefcase: markRaw(RiBriefcase4Fill),
  contacts: markRaw(RiContactsBook3Fill),
  chart: markRaw(RiPieChart2Fill),
  check: markRaw(RiCheckboxCircleFill),
};

const apps = ref<ApplicationItem[]>([]);
const loading = shallowRef(false);
const keyword = ref('');
// 弹窗只由「我的应用」入口控制；创建流程本身由应用领域组件承载，避免工作台耦合模板数据。
const createApplicationVisible = shallowRef(false);

const filteredApps = computed(() => {
  const kw = keyword.value.trim().toLowerCase();
  if (!kw) return apps.value;
  return apps.value.filter((app) => app.name.toLowerCase().includes(kw));
});

async function loadApps() {
  loading.value = true;
  try {
    const page = await listApplications({ limit: 100 });
    apps.value = page.items;
  } catch {
    // 无 applications:list 权限（普通成员未授权）等工作台场景回落空列表，
    // 不弹错误打断工作台渲染；按钮级能力由 capabilities 字段驱动
    apps.value = [];
  } finally {
    loading.value = false;
  }
}

// 创建空白应用（§15 前端契约）：异步提交——成功后刷新「我的应用」并放行
// 关闭弹窗；失败按 errCode 分支提示并保留弹窗填写内容
async function handleCreateBlank(draft: BlankApplicationDraft): Promise<boolean> {
  try {
    await createBlankApplication({ name: draft.name, icon: draft.icon });
    ElMessage.success('应用创建成功');
    await loadApps();
    return true;
  } catch (error) {
    if (error instanceof ApiError && error.errCode === ERROR_CODES.QUOTA_EXCEEDED) {
      ElMessage.error('应用数量已达套餐上限，请升级套餐或联系管理员');
      return false;
    }
    ElMessage.error(error instanceof Error ? error.message : '应用创建失败，请稍后重试');
    return false;
  }
}

onMounted(() => {
  // 编辑模式下不发起业务请求，避免工作台编排时的无效调用
  if (!props.editorMode) void loadApps();
});
</script>

<template>
  <DashboardWidgetFrame :title="widget.title">
    <template v-if="!props.editorMode" #actions>
      <div class="apps-widget__actions">
        <el-input
          v-model="keyword"
          placeholder="请输入名称搜索"
          :prefix-icon="RiSearchFill"
          clearable
        />
        <el-button type="primary" :icon="RiAddFill" @click="createApplicationVisible = true">
          新建应用
        </el-button>
      </div>
    </template>
    <div v-loading="loading" class="apps-widget">
      <el-empty
        v-if="!filteredApps.length"
        class="apps-widget__empty"
        description="暂无应用，点击「新建应用」开始"
        :image-size="56"
      />
      <el-button v-for="app in filteredApps" :key="app.id" class="apps-widget__item" text>
        <span class="apps-widget__icon" :class="`apps-widget__icon--${app.color}`">
          <component :is="iconByKey[app.icon] ?? iconByKey.bookmark" />
        </span>
        <span class="apps-widget__item-name" :title="app.name">{{ app.name }}</span>
      </el-button>
    </div>
  </DashboardWidgetFrame>
  <CreateApplicationDialog v-model="createApplicationVisible" :submit-blank="handleCreateBlank" />
</template>

<style scoped lang="scss">
.apps-widget {
  display: flex;
  align-items: flex-end;
  height: 100%;
  gap: 28px;

  &__empty {
    width: 100%;
    padding: 0;
    margin: 0;
  }

  &__actions {
    display: flex;
    width: 320px;
    gap: 8px;
  }

  &__item {
    display: inline-flex;
    max-width: 120px;
    flex-direction: column;
    height: auto;
    margin: 0;
    color: var(--el-text-color-primary);
    line-height: 1.5;

    &:hover {
      color: var(--el-color-primary);
    }
  }

  &__item-name {
    width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    margin-bottom: 8px;
    color: var(--el-color-white);
    border-radius: var(--el-border-radius-base);
    font-size: 20px;

    // 颜色键映射主题色变量（后端稳定枚举，禁止字面量色值）
    &--primary {
      background: var(--el-color-primary);
    }
  }
}
</style>
