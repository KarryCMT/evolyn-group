<script setup lang="ts">
import { RiAddFill, RiSearchFill } from '@remixicon/vue';
import { DashboardWidgetFrame } from '@evolyn.do/dashboard';
import { EvolynIconPicker } from '@evolyn.do/ui';
import { ApiError, ERROR_CODES } from '@evolyn.do/utils';
import { ElMessage } from 'element-plus';
import { computed, onMounted, ref, shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import { createBlankApplication, listApplications } from '~/api/applications';
import CreateApplicationDialog from '~/components/application/create/CreateApplicationDialog.vue';
import type { BlankApplicationDraft } from '~/components/application/create/BlankApplicationDialog.vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import type { ApplicationItem } from '~/types';

defineOptions({ name: 'AppsWidget' });
const props = withDefaults(
  defineProps<{
    widget: DashboardWidgetContent;
    editorMode?: boolean;
  }>(),
  { editorMode: false },
);
const router = useRouter();

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

/** 打开应用首页：路由参数使用可公开引用的稳定应用编码，不暴露内部主键。 */
function openApplication(app: ApplicationItem) {
  void router.push({ name: 'App', params: { appCode: app.code } });
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
      <!-- 原生按钮避免 Element Plus 包装插槽内容，保证图标与名称的固定纵向布局。 -->
      <button
        v-for="app in filteredApps"
        :key="app.id"
        class="apps-widget__item"
        type="button"
        @click="openApplication(app)"
      >
        <EvolynIconPicker
          class="apps-widget__icon"
          :model-value="app.icon"
          display-only
          :size="48"
        />
        <span class="apps-widget__item-name" :title="app.name">{{ app.name }}</span>
      </button>
    </div>
  </DashboardWidgetFrame>
  <CreateApplicationDialog v-model="createApplicationVisible" :submit-blank="handleCreateBlank" />
</template>

<style scoped lang="scss">
.apps-widget {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(112px, 1fr));
  align-content: start;
  height: 100%;
  gap: var(--el-space-4xl) var(--el-space-2xl);

  &__empty {
    width: 100%;
    padding: 0;
    margin: 0;
  }

  &__actions {
    display: flex;
    width: min(320px, calc(100vw - 132px));
    min-width: 0;
    gap: var(--el-space-md);

    :deep(.el-input) {
      min-width: 0;
    }

    :deep(.el-button) {
      flex: none;
    }
  }

  &__item {
    display: flex;
    box-sizing: border-box;
    width: 100%;
    height: 96px;
    min-width: 0;
    flex-direction: column;
    align-items: center;
    justify-content: flex-start;
    padding: var(--el-space-md);
    border: 0;
    color: var(--el-text-color-primary);
    line-height: 1.5;
    border-radius: var(--el-border-radius-base);
    cursor: pointer;
    background: transparent;
    font: inherit;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__item-name {
    width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__icon {
    flex: 0 0 48px;
    margin-bottom: var(--el-space-md);
  }
}

@media (max-width: 640px) {
  .apps-widget {
    grid-template-columns: repeat(auto-fill, minmax(96px, 1fr));
    gap: var(--el-space-3xl) var(--el-space-lg);

    &__item {
      height: 88px;
      padding: var(--el-space-sm);
    }

    &__icon {
      margin-bottom: var(--el-space-sm);
    }
  }
}

// 暗黑主题下主色浅阶未必是深色表面，悬停改用随主题切换的中性填充色。
:global(html.dark) .apps-widget__item:hover {
  background: var(--el-fill-color-light);
}
</style>
