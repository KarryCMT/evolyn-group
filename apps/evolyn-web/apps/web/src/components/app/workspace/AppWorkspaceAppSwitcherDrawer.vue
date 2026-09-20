<script setup lang="ts">
import type { AppItem } from '~/types';
import { EvolynIconPicker } from '@evolyn.do/ui';
import { RiCloseFill, RiHome5Fill, RiSearch2Line } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import { listApps } from '~/api/apps';

defineOptions({ name: 'AppWorkspaceAppSwitcherDrawer' });

const props = defineProps<{
  /** 当前工作区应用，用于抽屉列表中的选中态。 */
  activeAppCode: string;
}>();

const emit = defineEmits<{
  back: [];
  selectApp: [appCode: string];
}>();

const visible = defineModel<boolean>({ default: false });
const searchText = shallowRef('');
const apps = shallowRef<AppItem[]>([]);
const status = shallowRef<'loading' | 'ready' | 'error'>('loading');
let requestVersion = 0;

/** 搜索仅在已读取的可访问应用中执行，避免每个按键都触发一次网络请求。 */
const filteredApps = computed(() => {
  const keyword = searchText.value.trim().toLocaleLowerCase();
  if (!keyword) return apps.value;
  return apps.value.filter((app) => app.name.toLocaleLowerCase().includes(keyword));
});

async function loadApps() {
  const version = ++requestVersion;
  status.value = 'loading';
  const collected: AppItem[] = [];
  let cursor = '';

  try {
    // 应用数量受套餐配额约束，抽屉打开时逐页取齐，保障本地搜索结果完整。
    do {
      const page = await listApps({ status: 'active', limit: 100, cursor: cursor || undefined });
      if (version !== requestVersion) return;
      collected.push(...page.items.filter((app) => app.capabilities.view));
      cursor = page.nextCursor;
    } while (cursor);

    apps.value = collected;
    status.value = 'ready';
  } catch (error) {
    if (version !== requestVersion) return;
    console.warn('[app-switcher] load apps failed', error);
    status.value = 'error';
  }
}

function selectApp(app: AppItem) {
  visible.value = false;
  if (app.code !== props.activeAppCode) emit('selectApp', app.code);
}

watch(
  () => visible.value,
  (isVisible) => {
    if (!isVisible) return;
    searchText.value = '';
    void loadApps();
  },
);
</script>

<template>
  <el-drawer
    v-model="visible"
    class="app-workspace-app-switcher"
    direction="ltr"
    size="34%"
    :show-close="false"
    :lock-scroll="true"
    append-to-body
  >
    <template #header>
      <header class="app-workspace-app-switcher__header">
        <button class="app-workspace-app-switcher__back" type="button" @click="emit('back')">
          <RiHome5Fill aria-hidden="true" />
          <span>返回工作台</span>
        </button>
        <button
          class="app-workspace-app-switcher__close"
          type="button"
          aria-label="关闭应用列表"
          @click="visible = false"
        >
          <RiCloseFill aria-hidden="true" />
        </button>
      </header>
    </template>

    <section class="app-workspace-app-switcher__content" aria-labelledby="app-switcher-heading">
      <label class="app-workspace-app-switcher__search">
        <RiSearch2Line aria-hidden="true" />
        <input v-model="searchText" type="search" placeholder="请输入名称来搜索" aria-label="搜索应用">
      </label>

      <h1 id="app-switcher-heading" class="app-workspace-app-switcher__heading">
        我的应用
      </h1>

      <div v-if="status === 'loading'" class="app-workspace-app-switcher__hint">
        应用加载中…
      </div>
      <div v-else-if="status === 'error'" class="app-workspace-app-switcher__hint">
        <span>应用加载失败，请稍后重试</span>
        <button type="button" @click="loadApps">
          重新加载
        </button>
      </div>
      <div v-else-if="!filteredApps.length" class="app-workspace-app-switcher__hint">
        {{ searchText ? '未找到匹配的应用' : '暂无可访问的应用' }}
      </div>
      <nav v-else class="app-workspace-app-switcher__list" aria-label="我的应用">
        <button
          v-for="app in filteredApps"
          :key="app.code"
          class="app-workspace-app-switcher__item"
          :class="{ 'app-workspace-app-switcher__item--active': app.code === props.activeAppCode }"
          type="button"
          :aria-current="app.code === props.activeAppCode ? 'page' : undefined"
          @click="selectApp(app)"
        >
          <span class="app-workspace-app-switcher__item-icon" aria-hidden="true">
            <EvolynIconPicker :model-value="app.icon" display-only :size="28" />
          </span>
          <span class="app-workspace-app-switcher__item-name">{{ app.name }}</span>
        </button>
      </nav>
    </section>
  </el-drawer>
</template>

<style lang="scss">
/* 抽屉传送至 body，使用唯一块类避免覆盖其他 Element Plus 抽屉。 */
.app-workspace-app-switcher.el-drawer {
  min-width: 440px;
  max-width: 560px;
  background: var(--el-bg-color);
  box-shadow: var(--el-box-shadow-dark);
}

.app-workspace-app-switcher .el-drawer__header {
  padding: 0;
  margin: 0;
}

.app-workspace-app-switcher .el-drawer__body {
  padding: 0;
}

.app-workspace-app-switcher__header {
  position: relative;
  display: flex;
  align-items: center;
  height: 72px;
  padding: 0 32px;
}

.app-workspace-app-switcher__back,
.app-workspace-app-switcher__close,
.app-workspace-app-switcher__item,
.app-workspace-app-switcher__hint button {
  cursor: pointer;
  background: transparent;
  border: 0;
}

.app-workspace-app-switcher__back {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  padding: 6px;
  font-size: 17px;
  color: var(--el-color-primary);
  border-radius: var(--el-border-radius-medium);

  svg {
    width: 20px;
    height: 20px;
  }

  &:hover {
    background: var(--el-color-primary-light-9);
  }
}

.app-workspace-app-switcher__close {
  position: absolute;
  top: 20px;
  right: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  font-size: 22px;
  color: var(--el-text-color-primary);
  border-radius: var(--el-border-radius-medium);

  &:hover {
    background: var(--el-fill-color-light);
  }
}

.app-workspace-app-switcher__content {
  padding: 8px 32px 32px;
}

.app-workspace-app-switcher__search {
  display: flex;
  gap: 10px;
  align-items: center;
  height: 44px;
  padding: 0 14px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border-radius: var(--el-border-radius-medium);

  svg {
    flex: 0 0 auto;
    width: 20px;
    height: 20px;
  }

  input {
    width: 100%;
    min-width: 0;
    padding: 0;
    font: inherit;
    font-size: 16px;
    color: var(--el-text-color-primary);
    outline: 0;
    background: transparent;
    border: 0;

    &::placeholder {
      color: var(--el-text-color-placeholder);
    }
  }
}

.app-workspace-app-switcher__heading {
  margin: 28px 0 12px;
  font-size: 20px;
  font-weight: 650;
  line-height: 28px;
  color: var(--el-text-color-primary);
}

.app-workspace-app-switcher__list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.app-workspace-app-switcher__item {
  display: flex;
  gap: 12px;
  align-items: center;
  width: 100%;
  min-height: 52px;
  padding: 8px 12px;
  color: var(--el-text-color-primary);
  text-align: left;
  border-radius: var(--el-border-radius-medium);

  &:hover,
  &--active {
    background: var(--el-fill-color-light);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }
}

.app-workspace-app-switcher__back:focus-visible,
.app-workspace-app-switcher__close:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}

.app-workspace-app-switcher__item-icon {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
}

.app-workspace-app-switcher__item-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 16px;
  font-weight: 500;
  line-height: 24px;
  white-space: nowrap;
}

.app-workspace-app-switcher__hint {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: center;
  min-height: 96px;
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-secondary);
}

.app-workspace-app-switcher__hint button {
  color: var(--el-color-primary);
}

@media (width <= 900px) {
  .app-workspace-app-switcher.el-drawer {
    width: 440px !important;
  }
}
</style>
