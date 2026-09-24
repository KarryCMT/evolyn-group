<script setup lang="ts">
import {
  RiAiGenerate2,
  RiArrowDownSLine,
  RiCalendarLine,
  RiFileList3Line,
  RiSearchLine,
} from '@remixicon/vue';
import { computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from 'vue';
import type {
  IntelligentFieldOption,
  IntelligentNodeFieldValueSource,
  IntelligentSourceFieldGroup,
} from '../../../schema';
import {
  intelligentFieldRecommendationScore,
  isIntelligentFieldCompatible,
} from '../../../schema';

defineOptions({ name: 'IntelligentNodeFieldPicker' });

const props = defineProps<{
  modelValue: IntelligentNodeFieldValueSource | null;
  targetField: IntelligentFieldOption;
  groups: readonly IntelligentSourceFieldGroup[];
}>();

const emit = defineEmits<{
  'update:modelValue': [source: IntelligentNodeFieldValueSource];
}>();

const rootRef = useTemplateRef<HTMLElement>('rootRef');
const open = shallowRef(false);
const keyword = shallowRef('');
const collapsedGroups = shallowRef<ReadonlySet<string>>(new Set());

const compatibleGroups = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  return props.groups
    .map((group) => ({
      ...group,
      fields: group.fields.filter(
        (field) =>
          isIntelligentFieldCompatible(props.targetField, field) &&
          (!normalized ||
            `${field.label}${field.widgetName}`.toLocaleLowerCase().includes(normalized)),
      ),
    }))
    .filter((group) => group.fields.length > 0);
});

const recommendations = computed(() =>
  props.groups
    .flatMap((group) =>
      group.fields.map((field) => ({
        nodeId: group.nodeId,
        nodeName: group.nodeName,
        field,
        score: intelligentFieldRecommendationScore(props.targetField, field),
      })),
    )
    .filter((item) => item.score > 0)
    .sort((left, right) => right.score - left.score)
    .slice(0, 3),
);

const selected = computed(() => {
  const source = props.modelValue;
  if (!source) return null;
  for (const group of props.groups) {
    const field = group.fields.find((item) => item.widgetName === source.field);
    if (group.nodeId === source.nodeId && field) return { group, field };
  }
  return null;
});

function choose(nodeId: string, field: IntelligentFieldOption): void {
  emit('update:modelValue', { type: 'node-field', nodeId, field: field.widgetName });
  open.value = false;
  keyword.value = '';
}

function toggleGroup(nodeId: string): void {
  const next = new Set(collapsedGroups.value);
  next.has(nodeId) ? next.delete(nodeId) : next.add(nodeId);
  collapsedGroups.value = next;
}

function closeOnOutside(event: PointerEvent): void {
  if (!rootRef.value?.contains(event.target as Node)) open.value = false;
}

onMounted(() => document.addEventListener('pointerdown', closeOnOutside));
onBeforeUnmount(() => document.removeEventListener('pointerdown', closeOnOutside));
</script>

<template>
  <div ref="rootRef" class="node-field-picker">
    <button
      type="button"
      class="node-field-picker__control"
      :class="{ 'is-open': open }"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span :class="{ 'is-placeholder': !selected }">
        {{ selected ? `${selected.group.nodeName}—${selected.field.label}` : '请选择字段' }}
      </span>
      <RiArrowDownSLine aria-hidden="true" />
    </button>

    <div v-if="open" class="node-field-picker__menu" role="dialog" aria-label="选择节点字段">
      <label class="node-field-picker__search">
        <RiSearchLine aria-hidden="true" />
        <input v-model="keyword" placeholder="搜索" autocomplete="off">
      </label>

      <div class="node-field-picker__body">
        <section v-if="recommendations.length > 0 && !keyword" class="node-field-picker__group">
          <h4><RiAiGenerate2 />推荐字段</h4>
          <button
            v-for="item in recommendations"
            :key="`recommend-${item.nodeId}-${item.field.widgetName}`"
            type="button"
            @click="choose(item.nodeId, item.field)"
          >
            <RiCalendarLine v-if="item.field.valueKind === 'date'" />
            <RiFileList3Line v-else />
            <span>{{ item.nodeName }}—{{ item.field.label }}</span>
          </button>
        </section>

        <section v-for="group in compatibleGroups" :key="group.nodeId" class="node-field-picker__group">
          <button type="button" class="node-field-picker__group-title" @click="toggleGroup(group.nodeId)">
            <RiArrowDownSLine :class="{ 'is-collapsed': collapsedGroups.has(group.nodeId) }" />
            <RiFileList3Line />
            <strong>{{ group.nodeName }}</strong>
          </button>
          <template v-if="!collapsedGroups.has(group.nodeId)">
            <button
              v-for="field in group.fields"
              :key="`${group.nodeId}-${field.widgetName}`"
              type="button"
              class="node-field-picker__field"
              @click="choose(group.nodeId, field)"
            >
              <RiCalendarLine v-if="field.valueKind === 'date'" />
              <RiFileList3Line v-else />
              <span>{{ field.label }}</span>
            </button>
          </template>
        </section>
        <p v-if="compatibleGroups.length === 0">没有兼容的字段</p>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.node-field-picker {
  position: relative;
  min-width: 0;
  flex: 1;

  &__control {
    display: flex;
    width: 100%;
    height: 46px;
    padding: 0 15px;
    align-items: center;
    justify-content: space-between;
    color: #263247;
    background: #fff;
    border: 0;
    cursor: pointer;
    font: inherit;
    text-align: left;

    &.is-open { box-shadow: inset 0 0 0 1px #11b8ad; }
    .is-placeholder { color: #a0a8b5; }
    svg { width: 21px; height: 21px; flex: 0 0 auto; }
  }

  &__menu {
    position: absolute;
    z-index: 85;
    right: 0;
    bottom: calc(100% + 8px);
    width: min(690px, 70vw);
    background: #fff;
    border: 1px solid #e0e5ec;
    border-radius: 9px;
    box-shadow: 0 14px 36px rgb(31 43 61 / 18%);
  }

  &__search {
    display: flex;
    height: 50px;
    padding: 0 15px;
    align-items: center;
    gap: 10px;
    color: #596579;
    border-bottom: 1px solid #e1e5eb;

    svg { width: 21px; height: 21px; }
    input { min-width: 0; flex: 1; border: 0; outline: 0; font: inherit; font-size: 15px; }
  }

  &__body { max-height: 430px; padding: 9px 15px 14px; overflow-y: auto; }
  &__body > p { margin: 30px 0; color: #9aa3af; text-align: center; }

  &__group {
    display: grid;
    margin-bottom: 8px;

    h4 {
      display: flex;
      margin: 0;
      min-height: 36px;
      align-items: center;
      gap: 8px;
      color: #2998e9;
      font-size: 14px;
    }
    h4 svg { width: 19px; height: 19px; }

    > button:not(.node-field-picker__group-title) {
      display: flex;
      min-height: 38px;
      padding: 0 15px 0 76px;
      align-items: center;
      gap: 10px;
      color: #263247;
      background: transparent;
      border: 0;
      border-radius: 6px;
      cursor: pointer;
      font: inherit;
      text-align: left;

      &:hover { color: #247cf0; background: #f3f7fc; }
      svg { width: 18px; height: 18px; flex: 0 0 auto; }
    }
  }

  &__group-title {
    display: flex;
    min-height: 38px;
    padding: 0 15px;
    align-items: center;
    gap: 8px;
    color: #344055;
    background: transparent;
    border: 0;
    cursor: pointer;
    font: inherit;
    text-align: left;

    svg { width: 19px; height: 19px; transition: transform 0.16s ease; }
    svg.is-collapsed { transform: rotate(-90deg); }
  }
}
</style>
