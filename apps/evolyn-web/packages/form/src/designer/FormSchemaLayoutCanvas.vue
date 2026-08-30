<script setup lang="ts">
import { Delete } from '@element-plus/icons-vue';
import { EvolynScrollbar } from '@evolyn.do/ui';
import { ElButton, ElTabPane, ElTabs } from 'element-plus';
import { computed, reactive, type CSSProperties } from 'vue';
import Draggable from 'vuedraggable';
import type { FormItem, FormMultitabLayout, FormSchemaDocument } from '../schema/types';
import { isLayoutWidgetType } from '../schema/codec';
import { FORM_SCHEMA_DRAG_GROUP, type FormSchemaPaletteDrag } from './palette';
import type { FormLayoutTarget } from './useFormSchemaEditor';
import FormSchemaFieldCard from './FormSchemaFieldCard.vue';

/**
 * v2 布局画布：只渲染引用序列，不复制字段定义；跨列表拖拽通过语义事件回写编辑器，
 * 因而标签页未来可无差别承载 subform 等任意顶层字段。
 */
const props = defineProps<{ document: FormSchemaDocument; selectedKey: string }>();

const emit = defineEmits<{
  selectItem: [key: string];
  selectLayout: [name: string];
  copyItem: [item: FormItem];
  removeItem: [key: string];
  replaceReferences: [payload: { target: FormLayoutTarget; entries: unknown[] }];
  removeLayout: [layoutName: string];
}>();

const activeTabs = reactive<Record<string, string>>({});
const itemMap = computed(
  () => new Map(props.document.content.items.map((item) => [item.widget.widgetName, item])),
);
const layoutMap = computed(
  () => new Map(props.document.content.layout_fields.map((layout) => [layout.name, layout])),
);

function itemOf(reference: unknown): FormItem | undefined {
  return typeof reference === 'string' ? itemMap.value.get(reference) : undefined;
}

function layoutOf(reference: unknown): FormMultitabLayout | undefined {
  return typeof reference === 'string' ? layoutMap.value.get(reference) : undefined;
}

function referenceKey(reference: unknown): string {
  if (typeof reference === 'string') return reference;
  return (reference as Partial<FormSchemaPaletteDrag> | null)?.paletteType ?? '';
}

/** 画布与运行时共用 12 栅格；布局控件和标签页容器固定跨满整行。 */
function referenceStyle(reference: unknown): CSSProperties {
  const item = itemOf(reference);
  const width =
    item && !isLayoutWidgetType(item.widget.type) && item.lineWidth >= 1 && item.lineWidth <= 12
      ? item.lineWidth
      : 12;
  return { '--form-schema-field-span': width } as CSSProperties;
}

function replaceTop(entries: unknown[]): void {
  emit('replaceReferences', { target: { type: 'top' }, entries });
}

function replaceTab(layoutName: string, tabName: string, entries: unknown[]): void {
  emit('replaceReferences', {
    target: { type: 'tab', layoutName, tabName },
    entries,
  });
}

function activeTabName(layout: FormMultitabLayout): string {
  const current = activeTabs[layout.name];
  if (layout.container.some((tab) => tab.name === current)) return current;
  return layout.container[0]?.name ?? '';
}

function allowTabMove(event: { draggedContext?: { element?: unknown } }): boolean {
  const element = event.draggedContext?.element;
  return typeof element !== 'string' || !element.startsWith('_layout_');
}
</script>

<template>
  <EvolynScrollbar class="form-schema-layout-canvas">
    <Draggable
      :model-value="document.content.field_layout"
      :group="{ name: FORM_SCHEMA_DRAG_GROUP, pull: true, put: true }"
      :item-key="referenceKey"
      class="form-schema-layout-canvas__list"
      ghost-class="form-schema-layout-canvas__ghost"
      :animation="180"
      @update:model-value="replaceTop"
    >
      <template #item="{ element }">
        <div class="form-schema-layout-canvas__node" :style="referenceStyle(element)">
          <FormSchemaFieldCard
            v-if="itemOf(element)"
            :item="itemOf(element)!"
            :selected="selectedKey === element"
            @select="emit('selectItem', $event)"
            @copy="emit('copyItem', $event)"
            @remove="emit('removeItem', $event)"
          />

          <section
            v-else-if="layoutOf(element)"
            class="form-schema-layout-canvas__tabs"
            :class="{ 'is-active': selectedKey === element }"
            @click="emit('selectLayout', element)"
          >
            <header class="form-schema-layout-canvas__tabs-header">
              <strong>标签页</strong>
              <span class="form-schema-layout-canvas__tabs-hint">在右侧配置标签页属性</span>
              <el-button
                text
                type="danger"
                title="解散标签页"
                :icon="Delete"
                @click.stop="emit('removeLayout', element)"
              />
            </header>

            <el-tabs
              :model-value="activeTabName(layoutOf(element)!)"
              :type="layoutOf(element)!.tabStyle === 'style2' ? 'card' : undefined"
              @update:model-value="activeTabs[element] = String($event)"
            >
              <el-tab-pane
                v-for="tab in layoutOf(element)!.container"
                :key="tab.name"
                :name="tab.name"
                :label="tab.title"
              >
                <Draggable
                  :model-value="tab.field_layout"
                  :group="{ name: FORM_SCHEMA_DRAG_GROUP, pull: true, put: true }"
                  :item-key="referenceKey"
                  class="form-schema-layout-canvas__tab-list"
                  ghost-class="form-schema-layout-canvas__ghost"
                  :animation="180"
                  :move="allowTabMove"
                  @update:model-value="replaceTab(element, tab.name, $event)"
                >
                  <template #item="{ element: fieldReference }">
                    <div
                      class="form-schema-layout-canvas__node"
                      :style="referenceStyle(fieldReference)"
                      @click.stop
                    >
                      <FormSchemaFieldCard
                        v-if="itemOf(fieldReference)"
                        :item="itemOf(fieldReference)!"
                        :selected="selectedKey === fieldReference"
                        @select="emit('selectItem', $event)"
                        @copy="emit('copyItem', $event)"
                        @remove="emit('removeItem', $event)"
                      />
                    </div>
                  </template>
                </Draggable>
                <p
                  v-if="tab.field_layout.length === 0"
                  class="form-schema-layout-canvas__tab-empty"
                >
                  将字段拖入当前标签页
                </p>
              </el-tab-pane>
            </el-tabs>
          </section>

          <div v-else class="form-schema-layout-canvas__pending">正在添加字段…</div>
        </div>
      </template>
    </Draggable>
    <div v-if="document.content.field_layout.length === 0" class="form-schema-layout-canvas__empty">
      从左侧选择字段或标签页
    </div>
  </EvolynScrollbar>
</template>

<style scoped lang="scss">
.form-schema-layout-canvas {
  position: relative;
  box-sizing: border-box;
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  padding: var(--el-space-2xl);
  border-top: 1px solid var(--el-border-color);

  &__list {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: var(--el-space-lg);
    min-height: 100%;
    padding-bottom: var(--el-space-4xl);
  }
  &__node {
    grid-column: span var(--form-schema-field-span, 12);
    min-width: 0;
  }
  &__tabs {
    padding: var(--el-space-xl);
    margin-bottom: var(--el-space-lg);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);

    &.is-active {
      background: var(--el-color-primary-light-9);
      border-color: var(--el-color-primary);
    }
  }
  &__tabs-header {
    display: flex;
    gap: var(--el-space-md);
    align-items: center;
  }
  &__tabs-header {
    margin-bottom: var(--el-space-md);
  }
  &__tabs-hint {
    margin-left: auto;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-extra-small);
  }
  &__tab-list {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: var(--el-space-lg);
    min-height: 72px;
    padding: var(--el-space-md);
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-base);
  }
  &__tab-empty,
  &__empty {
    color: var(--el-text-color-secondary);
    text-align: center;
    pointer-events: none;
  }
  &__tab-empty {
    margin: calc(-52px) 0 28px;
    font-size: var(--el-font-size-extra-small);
  }
  &__empty {
    position: absolute;
    inset: var(--el-space-2xl);
    display: grid;
    place-items: center;
  }
  &__pending,
  &__ghost {
    grid-column: 1 / -1;
    min-height: 48px;
    margin-bottom: var(--el-space-lg);
    color: transparent;
    border: 1px dashed var(--el-color-primary);
    border-radius: var(--el-border-radius-base);
  }
}
</style>
