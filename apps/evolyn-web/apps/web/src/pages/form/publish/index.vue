<script setup lang="ts">
import { RiContactsBook3Fill, RiPieChart2Fill, RiShareForwardFill } from '@remixicon/vue';
import { computed, markRaw, type Component } from 'vue';
import { useRoute } from 'vue-router';

defineOptions({ name: 'FormPublishLayout' });

interface PublishNavigationItem {
  name: string;
  label: string;
  icon: Component;
  dividerBefore?: boolean;
}

const route = useRoute();
const publishNavigationItems: PublishNavigationItem[] = [
  { name: 'form-publish-members', label: '对成员发布', icon: markRaw(RiContactsBook3Fill) },
  { name: 'form-publish-public', label: '公开发布', icon: markRaw(RiShareForwardFill) },
  {
    name: 'form-publish-views',
    label: '视图',
    icon: markRaw(RiPieChart2Fill),
    dividerBefore: true,
  },
];

const routeParams = computed(() => ({
  appCode: String(route.params.appCode ?? ''),
  formCode: String(route.params.formCode ?? ''),
}));
</script>

<template>
  <section class="form-publish-layout" aria-label="表单发布">
    <aside class="form-publish-layout__sidebar" aria-label="表单发布菜单">
      <nav class="form-publish-layout__navigation">
        <template v-for="item in publishNavigationItems" :key="item.name">
          <div v-if="item.dividerBefore" class="form-publish-layout__divider" />
          <RouterLink
            class="form-publish-layout__navigation-item"
            :to="{ name: item.name, params: routeParams }"
          >
            <component :is="item.icon" />
            <span>{{ item.label }}</span>
          </RouterLink>
        </template>
      </nav>
    </aside>

    <main class="form-publish-layout__content">
      <RouterView />
    </main>
  </section>
</template>

<style scoped lang="scss">
.form-publish-layout {
  display: flex;
  min-height: 0;
  margin: 0 var(--el-space-md) var(--el-space-md);
  overflow: hidden;
  flex: 1;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);

  &__sidebar {
    width: 254px;
    min-width: 254px;
    padding: var(--el-space-md);
    border-right: 1px solid var(--el-border-color-lighter);
  }

  &__navigation {
    display: flex;
    height: 100%;
    flex-direction: column;
    gap: var(--el-space-xs);
  }

  &__navigation-item {
    display: flex;
    min-height: 44px;
    padding: 0 var(--el-space-lg);
    align-items: center;
    gap: var(--el-space-lg);
    color: var(--el-text-color-regular);
    border-radius: var(--el-border-radius-base);
    cursor: pointer;
    font-size: var(--el-font-size-base);
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
    margin: var(--el-space-md) var(--el-space-xs);
    background: var(--el-border-color-lighter);
  }

  &__content {
    min-width: 0;
    flex: 1;
    overflow: auto;
  }
}

@media (max-width: 760px) {
  .form-publish-layout {
    margin: 0 var(--el-space-xs) var(--el-space-xs);
    border-radius: var(--el-border-radius-large);

    &__sidebar {
      width: 196px;
      min-width: 196px;
    }
  }
}
</style>
