<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue';
import type { DashboardWidget } from '~/types/dashboard';

defineOptions({ name: 'WidgetInspector' });

const props = defineProps<{
  widget: DashboardWidget | null;
}>();
const emit = defineEmits<{
  update: [id: string, patch: Partial<Pick<DashboardWidget, 'title' | 'config'>>];
}>();

const title = shallowRef('');
const shortcutPlaceholder = shallowRef('');
const onboardingVariant = shallowRef<'guide' | 'rich-text' | 'carousel'>('guide');
const favoritesVariant = shallowRef<'favorites' | 'recent'>('favorites');
const hasTitle = computed(() => props.widget?.type !== 'greeting');

watch(
  () => props.widget,
  (widget) => {
    title.value = widget?.title ?? '';
    shortcutPlaceholder.value = readConfigText(widget, 'placeholder');
    onboardingVariant.value = readVariant(widget, ['guide', 'rich-text', 'carousel'], 'guide');
    favoritesVariant.value = readVariant(widget, ['favorites', 'recent'], 'favorites');
  },
  { immediate: true },
);

function updateTitle() {
  const value = title.value.trim();
  if (!props.widget || !value || value === props.widget.title) return;

  emit('update', props.widget.id, { title: value });
}

function updateConfig(key: string, value: string) {
  if (!props.widget) return;

  emit('update', props.widget.id, {
    config: { ...props.widget.config, [key]: value },
  });
}

function readConfigText(widget: DashboardWidget | null, key: string) {
  const value = widget?.config?.[key];
  return typeof value === 'string' ? value : '';
}

function readVariant<TVariant extends string>(
  widget: DashboardWidget | null,
  variants: readonly TVariant[],
  fallback: TVariant,
): TVariant {
  const value = widget?.config?.variant;
  return typeof value === 'string' && variants.includes(value as TVariant)
    ? (value as TVariant)
    : fallback;
}
</script>

<template>
  <aside class="widget-inspector" aria-label="组件设置">
    <template v-if="widget">
      <div class="widget-inspector__header">
        <strong class="widget-inspector__title">组件设置</strong>
        <span class="widget-inspector__type">{{ widget.type }}</span>
      </div>
      <el-scrollbar class="widget-inspector__scrollbar">
        <el-form class="widget-inspector__form" label-position="top">
          <el-form-item v-if="hasTitle" label="组件标题">
            <el-input v-model="title" maxlength="30" show-word-limit @change="updateTitle" />
          </el-form-item>

          <template v-if="widget.type === 'shortcut'">
            <el-form-item label="空状态文案">
              <el-input
                v-model="shortcutPlaceholder"
                maxlength="40"
                show-word-limit
                @change="updateConfig('placeholder', shortcutPlaceholder)"
              />
            </el-form-item>
          </template>

          <template v-else-if="widget.type === 'onboarding'">
            <el-form-item label="展示类型">
              <el-select
                v-model="onboardingVariant"
                @change="updateConfig('variant', onboardingVariant)"
              >
                <el-option label="新手引导" value="guide" />
                <el-option label="富文本" value="rich-text" />
                <el-option label="轮播图" value="carousel" />
              </el-select>
            </el-form-item>
          </template>

          <template v-else-if="widget.type === 'favorites'">
            <el-form-item label="展示内容">
              <el-select
                v-model="favoritesVariant"
                @change="updateConfig('variant', favoritesVariant)"
              >
                <el-option label="我的收藏" value="favorites" />
                <el-option label="最近使用" value="recent" />
              </el-select>
            </el-form-item>
          </template>

          <p v-else class="widget-inspector__hint">当前组件暂仅支持编辑标题与布局。</p>
        </el-form>
      </el-scrollbar>
    </template>
    <div v-else class="widget-inspector__empty">选择画布中的组件进行设置</div>
  </aside>
</template>

<style scoped lang="scss">
.widget-inspector {
  display: flex;
  flex: 0 0 280px;
  flex-direction: column;
  min-width: 0;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color-lighter);

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 56px;
    padding: 0 16px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__title {
    font-size: var(--el-font-size-base);
  }

  &__type {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }

  &__scrollbar {
    flex: 1;
    min-height: 0;
  }

  &__form {
    padding: 16px;
  }

  &__hint,
  &__empty {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
    line-height: 1.6;
  }

  &__empty {
    padding: 24px 16px;
  }
}
</style>
