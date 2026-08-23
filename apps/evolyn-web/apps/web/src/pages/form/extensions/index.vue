<script setup lang="ts">
import {
  RiAiGenerate2,
  RiCheckboxCircleFill,
  RiFileList2Fill,
  RiFileTextFill,
  RiNotification3Fill,
  RiPrinterFill,
  RiPushpin2Fill,
  RiSettings3Fill,
  RiWallet3Fill,
} from '@remixicon/vue';
import { computed, markRaw, type Component } from 'vue';
import { useRoute } from 'vue-router';

defineOptions({ name: 'FormExtensionsLayout' });

interface ExtensionNavigationItem {
  name: string;
  label: string;
  icon: Component;
  dividerBefore?: boolean;
}

const route = useRoute();
const extensionNavigationItems: ExtensionNavigationItem[] = [
  { name: 'form-extension-collaboration', label: '数据协作', icon: markRaw(RiSettings3Fill) },
  { name: 'form-extension-details', label: '数据详情', icon: markRaw(RiFileTextFill) },
  { name: 'form-extension-notifications', label: '推送提醒', icon: markRaw(RiNotification3Fill) },
  {
    name: 'form-extension-submit-prompt',
    label: '提交提示',
    icon: markRaw(RiCheckboxCircleFill),
    dividerBefore: true,
  },
  { name: 'form-extension-print', label: '打印模板', icon: markRaw(RiPrinterFill) },
  { name: 'form-extension-ai', label: '智能助手', icon: markRaw(RiAiGenerate2) },
  { name: 'form-extension-payment', label: '在线支付', icon: markRaw(RiWallet3Fill) },
  { name: 'form-extension-actions', label: '自定义按钮', icon: markRaw(RiPushpin2Fill) },
  { name: 'form-extension-push', label: '数据推送', icon: markRaw(RiFileList2Fill) },
];

const routeParams = computed(() => ({
  appCode: String(route.params.appCode ?? ''),
  formId: String(route.params.formId ?? ''),
}));
</script>

<template>
  <section class="form-extensions-layout" aria-label="表单扩展功能">
    <aside class="form-extensions-layout__sidebar" aria-label="扩展功能菜单">
      <nav class="form-extensions-layout__navigation">
        <template v-for="item in extensionNavigationItems" :key="item.name">
          <div v-if="item.dividerBefore" class="form-extensions-layout__divider" />
          <RouterLink
            class="form-extensions-layout__navigation-item"
            :to="{ name: item.name, params: routeParams }"
          >
            <component :is="item.icon" />
            <span>{{ item.label }}</span>
          </RouterLink>
        </template>
      </nav>
    </aside>

    <main class="form-extensions-layout__content">
      <RouterView />
    </main>
  </section>
</template>

<style scoped lang="scss">
.form-extensions-layout {
  display: flex;
  min-height: 0;
  margin: 0 8px 8px;
  overflow: hidden;
  flex: 1;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  box-shadow: var(--el-box-shadow-light);

  &__sidebar {
    width: 254px;
    min-width: 254px;
    padding: 10px;
    border-right: 1px solid var(--el-border-color-lighter);
  }

  &__navigation {
    display: flex;
    height: 100%;
    overflow-y: auto;
    flex-direction: column;
    gap: 4px;
  }

  &__navigation-item {
    display: flex;
    min-height: 44px;
    padding: 0 14px;
    align-items: center;
    gap: 12px;
    color: var(--el-text-color-regular);
    border-radius: var(--el-border-radius-base);
    cursor: pointer;
    font-size: 15px;
    font-weight: 550;
    text-decoration: none;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    svg {
      width: 20px;
      height: 20px;
      color: var(--el-text-color-secondary);
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &.router-link-active {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);

      svg {
        color: var(--el-color-primary);
      }
    }
  }

  &__divider {
    height: 1px;
    margin: 10px 4px;
    background: var(--el-border-color-lighter);
  }

  &__content {
    min-width: 0;
    flex: 1;
    overflow: auto;
  }
}

@media (max-width: 760px) {
  .form-extensions-layout {
    margin: 0 4px 4px;
    border-radius: 10px;

    &__sidebar {
      width: 196px;
      min-width: 196px;
    }
  }
}
</style>
