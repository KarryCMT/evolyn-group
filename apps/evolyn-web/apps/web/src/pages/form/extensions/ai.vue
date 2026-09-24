<script setup lang="ts">
import type {
  AssistantDraft,
  AssistantFormOption,
  AssistantListItem,
} from '~/components/form/assistant/assistant.types';
import {
  RiAddLine,
  RiFlowChart,
  RiLightbulbFlashLine,
  RiLinksLine,
  RiTimeLine,
} from '@remixicon/vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { listForms } from '~/api/form';
import AssistantCreateDialog from '~/components/form/assistant/AssistantCreateDialog.vue';
import AssistantList from '~/components/form/assistant/AssistantList.vue';
import { useFormWorkspaceContext } from '../workspace-context';

defineOptions({ name: 'FormAssistantPage' });

const { detail } = useFormWorkspaceContext();
const route = useRoute();
const router = useRouter();

const createDialogVisible = shallowRef(false);
const formsLoading = shallowRef(false);
const formOptions = shallowRef<AssistantFormOption[]>([]);
const assistants = shallowRef<AssistantListItem[]>([]);

const currentFormCode = computed(() => detail.value?.code ?? '');

watch(
  () => detail.value?.appId,
  (appId) => {
    const current = detail.value;
    if (current) {
      // 先呈现当前表单，网络目录返回后再替换完整选项；弹窗不需要等待请求完成。
      formOptions.value = [
        { code: current.code, name: current.name, formType: current.formType },
      ];
    }
    if (!appId) return;
    void loadForms(appId);
  },
  { immediate: true },
);

async function loadForms(appId: number): Promise<void> {
  formsLoading.value = true;
  try {
    const response = await listForms({ appId, limit: 100 });
    formOptions.value = response.items.map((item) => ({
      code: item.code,
      name: item.name,
      formType: item.formType,
    }));
  } catch {
    // 扩展功能页仍可基于当前表单完成前端配置，不因可选表单目录加载失败而阻塞。
    formOptions.value = detail.value
      ? [
          {
            code: detail.value.code,
            name: detail.value.name,
            formType: detail.value.formType,
          },
        ]
      : [];
    ElMessage.warning('表单目录加载失败，已仅保留当前表单');
  } finally {
    formsLoading.value = false;
  }
}

function openCreateDialog(): void {
  createDialogVisible.value = true;
}

function openDesigner(
  draft: AssistantDraft,
  assistantId: string = globalThis.crypto.randomUUID(),
): void {
  // 创建接口落地前由路由查询参数携带草稿摘要，刷新占位设计器仍能还原标题与触发信息。
  void router.push({
    name: 'form-assistant-designer',
    params: {
      appCode: String(route.params.appCode ?? ''),
      formCode: String(route.params.formCode ?? ''),
      assistantId,
    },
    query: {
      name: draft.name,
      triggerType: draft.triggerType,
      triggerFormCode: draft.triggerFormCode,
      triggerFormName: draft.triggerFormName,
      tags: draft.tags,
    },
  });
}

function editAssistant(item: AssistantListItem): void {
  openDesigner({
    name: item.name,
    triggerType: item.triggerType,
    triggerFormCode: item.triggerFormCode,
    triggerFormName: item.triggerFormName,
    tags: [...item.tags],
  }, item.id);
}

function copyAssistant(item: AssistantListItem): void {
  assistants.value = [
    ...assistants.value,
    {
      ...item,
      id: globalThis.crypto.randomUUID(),
      name: `${item.name} 副本`,
      enabled: false,
    },
  ];
  ElMessage.success('已复制智能助手 Pro');
}

function toggleAssistant(item: AssistantListItem): void {
  assistants.value = assistants.value.map((candidate) =>
    candidate.id === item.id ? { ...candidate, enabled: !candidate.enabled } : candidate,
  );
  ElMessage.success(item.enabled ? '智能助手 Pro 已停用' : '智能助手 Pro 已启用');
}

async function removeAssistant(item: AssistantListItem): Promise<void> {
  try {
    await ElMessageBox.confirm(`删除后无法恢复，确定删除“${item.name}”吗？`, '删除智能助手 Pro', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
      confirmButtonClass: 'assistant-delete-confirm',
    });
    assistants.value = assistants.value.filter((candidate) => candidate.id !== item.id);
    ElMessage.success('智能助手 Pro 已删除');
  } catch {
    // 用户取消删除时保持列表原状。
  }
}

function showHelp(): void {
  ElMessage.info('智能助手 Pro 帮助中心正在建设中');
}
</script>

<template>
  <section class="form-assistant-page" aria-label="智能助手 Pro">
    <header class="form-assistant-page__header">
      <div class="form-assistant-page__heading-copy">
        <h1>智能助手 Pro</h1>
        <p>实现自动同步更新表单数据、自动发送通知等智能化操作</p>
        <button type="button" @click="showHelp">
          帮助文档
        </button>
      </div>
    </header>

    <div class="form-assistant-page__content">
      <div class="form-assistant-page__toolbar">
        <ElButton type="primary" size="large" :icon="RiAddLine" @click="openCreateDialog">
          新建智能助手 Pro
        </ElButton>
        <span v-if="assistants.length > 0">共 {{ assistants.length }} 个智能助手</span>
      </div>

      <AssistantList
        v-if="assistants.length > 0"
        :items="assistants"
        @edit="editAssistant"
        @copy="copyAssistant"
        @toggle="toggleAssistant"
        @remove="removeAssistant"
      />

      <div v-else class="form-assistant-page__empty">
        <div class="form-assistant-page__guide-grid">
          <button type="button" @click="showHelp">
            <span class="form-assistant-page__guide-icon is-light"><RiLightbulbFlashLine /></span>
            <span>
              <small>快速了解</small>
              <strong>什么是智能助手 Pro？</strong>
            </span>
            <RiLinksLine class="form-assistant-page__guide-link" />
          </button>
          <button type="button" @click="showHelp">
            <span class="form-assistant-page__guide-icon is-flow"><RiFlowChart /></span>
            <span>
              <small>场景指南</small>
              <strong>智能助手 Pro 用在哪些业务场景？</strong>
            </span>
            <RiTimeLine class="form-assistant-page__guide-link" />
          </button>
        </div>
      </div>

      <p class="form-assistant-page__note">
        注：此处仅显示包含当前表单的智能助手 Pro。你可以在「应用管理后台」中管理应用内所有智能助手 Pro。
      </p>
    </div>

    <AssistantCreateDialog
      v-model="createDialogVisible"
      :forms="formOptions"
      :default-form-code="currentFormCode"
      :loading-forms="formsLoading"
      @create="openDesigner"
    />
  </section>
</template>

<style scoped lang="scss">
.form-assistant-page {
  --assistant-accent: #00afa2;
  --assistant-accent-hover: #08baad;

  min-height: 100%;
  color: #172033;
  background: #fff;

  &__header {
    min-height: 64px;
    padding: 0 32px;
    display: flex;
    align-items: center;
    border-bottom: 1px solid #e7eaf0;
  }

  &__heading-copy {
    display: flex;
    align-items: baseline;
    gap: 12px;

    h1,
    p {
      margin: 0;
    }

    h1 {
      font-size: 18px;
      font-weight: 650;
      line-height: 28px;
    }

    p {
      color: #7a8596;
      font-size: 14px;
    }

    button {
      padding: 0;
      color: var(--assistant-accent);
      background: transparent;
      border: 0;
      cursor: pointer;
      font: inherit;
    }
  }

  &__content {
    padding: 28px 32px 36px;
  }

  &__toolbar {
    display: flex;
    min-height: 48px;
    align-items: center;
    justify-content: space-between;
    gap: 16px;

    > span {
      color: #8a94a3;
      font-size: 13px;
    }
  }

  &__empty {
    margin-top: 46px;
    padding: 32px 44px;
    border-top: 1px solid #e2e6ec;
    border-bottom: 1px solid #e2e6ec;
  }

  &__guide-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;

    > button {
      display: flex;
      min-height: 96px;
      padding: 18px 22px;
      align-items: center;
      gap: 16px;
      color: #172033;
      text-align: left;
      background: #f5f8fb;
      border: 1px solid #edf0f4;
      border-radius: 8px;
      cursor: pointer;
      transition:
        border-color 0.18s ease,
        transform 0.18s ease,
        box-shadow 0.18s ease;

      &:hover {
        border-color: #b8e5df;
        box-shadow: 0 8px 20px rgb(32 70 86 / 8%);
        transform: translateY(-1px);
      }

      > span:nth-child(2) {
        display: flex;
        min-width: 0;
        flex: 1;
        flex-direction: column;
        gap: 4px;
      }

      small {
        color: #8390a1;
        font-size: 13px;
      }

      strong {
        color: #2372e5;
        font-size: 17px;
        font-weight: 600;
      }
    }
  }

  &__guide-icon {
    display: inline-flex;
    width: 52px;
    height: 52px;
    flex: 0 0 52px;
    align-items: center;
    justify-content: center;
    border-radius: 50%;

    svg {
      width: 30px;
      height: 30px;
    }

    &.is-light {
      color: #f4ad1f;
      background: #fff2c9;
    }

    &.is-flow {
      color: #4b91ef;
      background: #e7f1ff;
    }
  }

  &__guide-link {
    width: 22px;
    height: 22px;
    color: #a8b1bd;
  }

  &__note {
    margin: 26px 0 0;
    color: #778294;
    font-size: 14px;
    line-height: 22px;
  }
}

:deep(.form-assistant-page__toolbar .el-button--primary) {
  --el-button-bg-color: var(--assistant-accent);
  --el-button-border-color: var(--assistant-accent);
  --el-button-hover-bg-color: var(--assistant-accent-hover);
  --el-button-hover-border-color: var(--assistant-accent-hover);
}

@media (max-width: 820px) {
  .form-assistant-page {
    &__heading-copy {
      padding: 14px 0;
      align-items: flex-start;
      flex-direction: column;
      gap: 4px;
    }

    &__content {
      padding: 22px;
    }

    &__empty {
      padding: 24px 0;
    }

    &__guide-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style>
