<script setup lang="ts">
import { Delete } from '@element-plus/icons-vue';
import { EvolynScrollbar } from '@evolyn.do/ui';
import { ElIcon, ElPopconfirm, ElTabPane, ElTabs } from 'element-plus';
import { computed, reactive, type CSSProperties } from 'vue';
import Draggable from 'vuedraggable';
import type {
  FormItem,
  FormMultitabLayout,
  FormSchemaDocument,
  SubformWidget,
} from '../schema/types';
import { isLayoutWidgetType } from '../schema/codec';
import { FORM_LAYOUT_LINE_WIDTH } from '../schema/dictionary';
import { FORM_SCHEMA_DRAG_GROUP, type FormSchemaPaletteDrag } from './palette';
import type { FormLayoutTarget } from './useFormSchemaEditor';
import FormSchemaFieldCard from './FormSchemaFieldCard.vue';
import FormSchemaSubformCard from './FormSchemaSubformCard.vue';

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
  selectSubformItem: [parentKey: string, childKey: string];
  copySubformItem: [parentKey: string, childKey: string];
  removeSubformItem: [parentKey: string, childKey: string];
  replaceSubformItems: [parentKey: string, entries: unknown[]];
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
const dragListStyle = computed(
  () =>
    ({
      '--form-schema-drag-span': FORM_LAYOUT_LINE_WIDTH[props.document.content.layout],
    }) as CSSProperties,
);

/**
 * 外层画布与子表格都会接收素材面板，但外层不能在事件目标已落入子表格时接管克隆。
 * `move` 在插入候选项之后才触发，必须在 Sortable 的 `put` 阶段提前拒绝。
 */
const canvasDragGroup = {
  name: FORM_SCHEMA_DRAG_GROUP,
  pull: true,
  put: (_to: unknown, _from: unknown, _dragged: unknown, event: unknown): boolean => {
    const target = (event as { target?: EventTarget | null } | null)?.target;
    return !(target instanceof Element && target.closest('.form-schema-subform-card__columns'));
  },
};

function itemOf(reference: unknown): FormItem | undefined {
  return typeof reference === 'string' ? itemMap.value.get(reference) : undefined;
}

function subformItemOf(reference: unknown): FormItem<SubformWidget> | undefined {
  const item = itemOf(reference);
  return item?.widget.type === 'subform' ? (item as FormItem<SubformWidget>) : undefined;
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
  const paletteType = (reference as Partial<FormSchemaPaletteDrag> | null)?.paletteType;
  const width =
    item && !isLayoutWidgetType(item.widget.type) && item.lineWidth >= 1 && item.lineWidth <= 12
      ? item.lineWidth
      : typeof paletteType === 'string' && !isLayoutWidgetType(paletteType)
        ? FORM_LAYOUT_LINE_WIDTH[props.document.content.layout]
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

function activeTabName(layout: FormMultitabLayout): string | undefined {
  const current = activeTabs[layout.name];
  if (layout.container.some((tab) => tab.name === current)) return current;
  return layout.container[0]?.name ?? '';
}

function allowTabMove(event: { draggedContext?: { element?: unknown } }): boolean {
  const element = event.draggedContext?.element;
  return typeof element !== 'string' || !element.startsWith('_layout_');
}

/**
 * 素材字段可投放到画布，也可投放到子表单。二者是嵌套 Sortable，若外层继续接收
 * 指针已进入子表单列区的素材克隆，会与内层反复交换占位节点，造成拖拽抖动。
 */
function allowCanvasMove(event: { related?: Element | null }): boolean {
  return !event.related?.closest('.form-schema-subform-card__columns');
}
</script>

<template>
  <EvolynScrollbar class="form-schema-layout-canvas">
    <Draggable
      :model-value="document.content.field_layout"
      :group="canvasDragGroup"
      :item-key="referenceKey"
      class="form-schema-layout-canvas__list"
      :style="dragListStyle"
      ghost-class="form-schema-layout-canvas__ghost"
      :animation="180"
      filter=".form-schema-subform-card__table, .form-schema-subform-card__table *"
      :prevent-on-filter="false"
      :move="allowCanvasMove"
      @update:model-value="replaceTop"
    >
      <template #item="{ element }">
        <div class="form-schema-layout-canvas__node" :style="referenceStyle(element)">
          <FormSchemaSubformCard
            v-if="subformItemOf(element)"
            :item="subformItemOf(element)!"
            :selected-key="selectedKey"
            @select="emit('selectItem', $event)"
            @select-child="(parentKey, childKey) => emit('selectSubformItem', parentKey, childKey)"
            @copy="emit('copyItem', $event)"
            @remove="emit('removeItem', $event)"
            @replace-children="
              (parentKey, entries) => emit('replaceSubformItems', parentKey, entries)
            "
            @copy-child="(parentKey, childKey) => emit('copySubformItem', parentKey, childKey)"
            @remove-child="(parentKey, childKey) => emit('removeSubformItem', parentKey, childKey)"
          />
          <FormSchemaFieldCard
            v-else-if="itemOf(element)"
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
            <div class="form-schema-layout-canvas__tabs-actions">
              <el-popconfirm
                placement="bottom-end"
                :width="240"
                hide-icon
                confirm-button-type="danger"
                title="确定删除标签页布局？其中字段将移回表单画布"
                cancel-button-text="取消"
                confirm-button-text="删除"
                @confirm="emit('removeLayout', element)"
              >
                <template #reference>
                  <button
                    class="form-schema-layout-canvas__tabs-delete"
                    type="button"
                    title="删除标签页"
                    aria-label="删除标签页"
                    @click.stop
                  >
                    <el-icon><Delete /></el-icon>
                  </button>
                </template>
              </el-popconfirm>
            </div>

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
                  :group="canvasDragGroup"
                  :item-key="referenceKey"
                  class="form-schema-layout-canvas__tab-list"
                  :style="dragListStyle"
                  ghost-class="form-schema-layout-canvas__ghost"
                  :animation="180"
                  filter=".form-schema-subform-card__table, .form-schema-subform-card__table *"
                  :prevent-on-filter="false"
                  :move="allowTabMove"
                  @update:model-value="replaceTab(element, tab.name, $event)"
                >
                  <template #item="{ element: fieldReference }">
                    <div
                      class="form-schema-layout-canvas__node"
                      :style="referenceStyle(fieldReference)"
                      @click.stop
                    >
                      <FormSchemaSubformCard
                        v-if="subformItemOf(fieldReference)"
                        :item="subformItemOf(fieldReference)!"
                        :selected-key="selectedKey"
                        @select="emit('selectItem', $event)"
                        @select-child="
                          (parentKey, childKey) => emit('selectSubformItem', parentKey, childKey)
                        "
                        @copy="emit('copyItem', $event)"
                        @remove="emit('removeItem', $event)"
                        @replace-children="
                          (parentKey, entries) => emit('replaceSubformItems', parentKey, entries)
                        "
                        @copy-child="
                          (parentKey, childKey) => emit('copySubformItem', parentKey, childKey)
                        "
                        @remove-child="
                          (parentKey, childKey) => emit('removeSubformItem', parentKey, childKey)
                        "
                      />
                      <FormSchemaFieldCard
                        v-else-if="itemOf(fieldReference)"
                        :item="itemOf(fieldReference)!"
                        :selected="selectedKey === fieldReference"
                        @select="emit('selectItem', $event)"
                        @copy="emit('copyItem', $event)"
                        @remove="emit('removeItem', $event)"
                      />
                    </div>
                  </template>
                  <template #footer>
                    <p
                      v-if="tab.field_layout.length === 0"
                      class="form-schema-layout-canvas__tab-empty"
                    >
                      将字段拖入当前标签页
                    </p>
                  </template>
                </Draggable>
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
    grid-auto-rows: max-content;
    column-gap: var(--el-space-lg);
    row-gap: 0;
    align-content: start;
    min-height: 100%;
    padding-bottom: var(--el-space-4xl);
  }
  &__node {
    grid-column: span var(--form-schema-field-span, 12);
    min-width: 0;
  }
  &__tabs {
    position: relative;
    padding: var(--el-space-xl);
    background: var(--el-bg-color);

    &.is-active {
      background: var(--el-color-primary-light-9);
      border-color: var(--el-color-primary);
    }
  }
  &__tabs:hover &__tabs-actions,
  &__tabs.is-active &__tabs-actions {
    pointer-events: auto;
    opacity: 1;
  }
  &__tabs-actions {
    position: absolute;
    top: var(--el-space-xs);
    right: var(--el-space-xl);
    z-index: 1;
    display: flex;
    overflow: hidden;
    pointer-events: none;
    background: var(--el-bg-color);
    border-radius: var(--el-border-radius-base);
    box-shadow: var(--el-box-shadow);
    opacity: 0;
  }
  &__tabs-delete {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    background: transparent;
    border: 0;
  }
  &__tab-list {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    grid-auto-rows: max-content;
    column-gap: var(--el-space-lg);
    row-gap: 0;
    align-content: start;
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
    display: grid;
    grid-column: 1 / -1;
    min-height: 46px;
    margin: 0;
    place-items: center;
    font-size: var(--el-font-size-extra-small);
  }
  &__empty {
    position: absolute;
    inset: var(--el-space-2xl);
    display: grid;
    place-items: center;
  }
  &__pending {
    box-sizing: border-box;
    min-height: 64px;
    color: transparent;
    border: 1px dashed var(--el-color-primary);
    border-radius: var(--el-border-radius-base);
  }
}

// 跨列表拖拽时占位节点来自素材面板，不带本组件的 scoped 属性，因此必须全局命中。
:global(.form-schema-layout-canvas__ghost) {
  box-sizing: border-box !important;
  display: block !important;
  grid-column: span var(--form-schema-field-span, var(--form-schema-drag-span, 12)) !important;
  width: auto !important;
  min-width: 0 !important;
  min-height: 64px !important;
  padding: 0 !important;
  overflow: hidden !important;
  color: transparent !important;
  font-size: 0 !important;
  background: transparent !important;
  border: 1px dashed var(--el-color-primary) !important;
  border-radius: var(--el-border-radius-base) !important;
  box-shadow: none !important;
  opacity: 1 !important;
}

:global(.form-schema-layout-canvas__ghost > *) {
  visibility: hidden !important;
}
</style>
