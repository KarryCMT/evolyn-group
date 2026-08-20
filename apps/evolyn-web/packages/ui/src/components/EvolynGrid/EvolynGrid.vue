<script setup lang="ts">
import type { GridStack as GridStackInstance } from 'gridstack';
import { GridStack, type GridStackOptions } from 'gridstack/dist/vue';
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import type { EvolynGridComponents, EvolynGridEmits, EvolynGridItem, EvolynGridOptions } from './EvolynGrid.types';
import { mergeGridLayout, toGridStackWidgets } from './useEvolynGrid';

defineOptions({ name: 'EvolynGrid' });

const props = withDefaults(defineProps<{
  modelValue: EvolynGridItem[];
  options?: EvolynGridOptions;
  components: EvolynGridComponents;
  editable?: boolean;
}>(), {
  options: () => ({}),
  editable: false,
});

const emit = defineEmits<EvolynGridEmits>();

const gridComponent = ref<{ getGrid: () => GridStackInstance | null } | null>(null);
const isApplyingExternalLayout = ref(false);

const gridOptions = computed<GridStackOptions>(() => ({
  column: 12,
  cellHeight: 72,
  margin: 12,
  animate: true,
  ...props.options,
  children: toGridStackWidgets(props.modelValue),
}));

function getGrid(): GridStackInstance | null {
  return gridComponent.value?.getGrid() ?? null;
}

function emitLayoutChange(changedItems: Array<Partial<EvolynGridItem> & { id?: string }>) {
  if (isApplyingExternalLayout.value) return;

  const layout = mergeGridLayout(props.modelValue, changedItems);
  emit('update:modelValue', layout);
  emit('layout-change', layout);
}

function syncEditable(editable: boolean) {
  const grid = getGrid();
  if (!grid) return;
  if (editable) grid.enable();
  else grid.disable();
}

watch(() => props.editable, syncEditable);

watch(
  () => props.modelValue,
  async (items) => {
    const grid = getGrid();
    if (!grid) return;

    isApplyingExternalLayout.value = true;
    grid.load(toGridStackWidgets(items));
    await nextTick();
    isApplyingExternalLayout.value = false;
  },
  { deep: 1 },
);

onBeforeUnmount(() => {
  // 路由离开时释放 GridStack 的监听器，DOM 由 Vue 负责移除。
  getGrid()?.destroy(false);
});

onMounted(async () => {
  await nextTick();
  syncEditable(props.editable);
  emit('ready');
});

defineExpose({ getGrid });
</script>

<template>
  <GridStack
    ref="gridComponent"
    class="evolyn-grid"
    :options="gridOptions"
    :components="components"
    @change="(_event, items) => emitLayoutChange(items)"
    @added="(_event, items) => emitLayoutChange(items)"
    @removed="(_event, items) => emitLayoutChange(items)"
    @dragstop="() => emit('layout-change', modelValue)"
    @resizestop="() => emit('layout-change', modelValue)"
  >
    <template #empty>
      <slot name="empty" />
    </template>
  </GridStack>
</template>

<style lang="scss">
@use './EvolynGrid.scss' as *;
</style>
