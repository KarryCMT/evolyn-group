<script setup lang="ts">
import {
  RiAddLine,
  RiArrowDownSLine,
  RiArrowUpSLine,
  RiDeleteBinLine,
  RiEditBoxLine,
  RiFileCopyLine,
  RiFunctionLine,
  RiGitBranchLine,
  RiGlobalLine,
  RiNotification3Line,
} from '@remixicon/vue';
import {
  type Component,
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  shallowRef,
  useTemplateRef,
  watch,
} from 'vue';
import type { IntelligentActionType, IntelligentPosition } from '../../schema';
import { intelligentNodeTemplates } from '../../mock/nodeTemplates';
import { ACTION_COLORS } from '../../util';
import { clampInsertMenuPosition } from './position';

defineOptions({ name: 'IntelligentInsertMenu' });

const props = defineProps<{
  position: IntelligentPosition;
}>();
const emit = defineEmits<{
  select: [actionType: IntelligentActionType];
}>();
const visible = defineModel<boolean>({ default: false });

const menuRef = useTemplateRef<HTMLElement>('menuRef');
const showMore = shallowRef(false);
const renderedPosition = shallowRef<IntelligentPosition>({ x: 12, y: 12 });
const primaryTypes: readonly IntelligentActionType[] = [
  'create-record',
  'update-record',
  'delete-record',
  'data-transform',
];
const iconByType: Record<IntelligentActionType, Component> = {
  'create-record': RiAddLine,
  'update-record': RiEditBoxLine,
  'delete-record': RiDeleteBinLine,
  'data-transform': RiFunctionLine,
  'send-notification': RiNotification3Line,
  'http-request': RiGlobalLine,
  condition: RiGitBranchLine,
};

const menuStyle = computed(() => ({
  left: `${renderedPosition.value.x}px`,
  top: `${renderedPosition.value.y}px`,
}));
const primaryTemplates = computed(() =>
  primaryTypes.flatMap((type) => {
    const template = intelligentNodeTemplates.find((item) => item.type === type);
    return template ? [template] : [];
  }),
);
const moreTemplates = computed(() =>
  intelligentNodeTemplates.filter((item) => !primaryTypes.includes(item.type)),
);

function select(actionType: IntelligentActionType): void {
  emit('select', actionType);
  visible.value = false;
}

function closeOnOutside(event: PointerEvent): void {
  if (!visible.value || menuRef.value?.contains(event.target as Node)) return;
  visible.value = false;
}

function closeOnEscape(event: KeyboardEvent): void {
  if (!visible.value || event.key !== 'Escape') return;
  event.preventDefault();
  visible.value = false;
}

function updateMenuPosition(): void {
  const menu = menuRef.value;
  const viewport = menu?.parentElement;
  if (!menu || !viewport) return;

  renderedPosition.value = clampInsertMenuPosition(props.position, {
    viewportWidth: viewport.clientWidth,
    viewportHeight: viewport.clientHeight,
    menuWidth: menu.offsetWidth,
    menuHeight: menu.offsetHeight,
  });
}

let resizeObserver: ResizeObserver | null = null;
let positionFrame = 0;

function scheduleMenuPosition(): void {
  cancelAnimationFrame(positionFrame);
  positionFrame = requestAnimationFrame(updateMenuPosition);
}

function observeMenuBounds(): void {
  resizeObserver?.disconnect();
  const menu = menuRef.value;
  const viewport = menu?.parentElement;
  if (!menu || !viewport) return;
  resizeObserver?.observe(menu);
  resizeObserver?.observe(viewport);
  scheduleMenuPosition();
}

watch(
  [visible, () => props.position.x, () => props.position.y],
  async ([nextVisible]) => {
    if (!nextVisible) {
      showMore.value = false;
      resizeObserver?.disconnect();
      return;
    }
    await nextTick();
    observeMenuBounds();
  },
  { flush: 'post', immediate: true },
);
watch(showMore, async () => {
  if (!visible.value) return;
  await nextTick();
  scheduleMenuPosition();
});

onMounted(() => {
  resizeObserver = new ResizeObserver(scheduleMenuPosition);
  if (visible.value) observeMenuBounds();
  document.addEventListener('pointerdown', closeOnOutside);
  document.addEventListener('keydown', closeOnEscape);
});

onBeforeUnmount(() => {
  cancelAnimationFrame(positionFrame);
  resizeObserver?.disconnect();
  resizeObserver = null;
  document.removeEventListener('pointerdown', closeOnOutside);
  document.removeEventListener('keydown', closeOnEscape);
});
</script>

<template>
  <section
    v-if="visible"
    ref="menuRef"
    class="intelligent-insert-menu"
    :style="menuStyle"
    role="dialog"
    aria-label="添加下级节点"
    @pointerdown.stop
  >
    <button type="button" class="intelligent-insert-menu__paste" disabled>
      <RiFileCopyLine aria-hidden="true" />
      粘贴节点
    </button>

    <p class="intelligent-insert-menu__group-title">数据处理</p>
    <div class="intelligent-insert-menu__grid">
      <button
        v-for="item in primaryTemplates"
        :key="item.type"
        type="button"
        class="intelligent-insert-menu__item"
        @click="select(item.type)"
      >
        <component
          :is="iconByType[item.type]"
          :style="{ color: ACTION_COLORS[item.type].foreground }"
          aria-hidden="true"
        />
        <span>{{ item.name }}</span>
      </button>
    </div>

    <button
      type="button"
      class="intelligent-insert-menu__more"
      :aria-expanded="showMore"
      @click="showMore = !showMore"
    >
      <RiArrowUpSLine v-if="showMore" aria-hidden="true" />
      <RiArrowDownSLine v-else aria-hidden="true" />
      更多节点
    </button>

    <div v-if="showMore" class="intelligent-insert-menu__grid intelligent-insert-menu__grid--more">
      <button
        v-for="item in moreTemplates"
        :key="item.type"
        type="button"
        class="intelligent-insert-menu__item"
        @click="select(item.type)"
      >
        <component
          :is="iconByType[item.type]"
          :style="{ color: ACTION_COLORS[item.type].foreground }"
          aria-hidden="true"
        />
        <span>{{ item.name }}</span>
      </button>
    </div>
  </section>
</template>

<style scoped lang="scss">
.intelligent-insert-menu {
  position: absolute;
  z-index: 20;
  width: min(420px, calc(100% - 24px));
  max-height: calc(100% - 24px);
  padding: 16px;
  overflow-y: auto;
  overscroll-behavior: contain;
  color: #172033;
  background: #fff;
  border: 1px solid #e1e6ee;
  border-radius: 11px;
  box-shadow: 0 12px 34px rgb(31 43 61 / 17%);
  box-sizing: border-box;

  button { font: inherit; }

  &__paste {
    display: flex;
    width: 100%;
    height: 42px;
    padding: 0 14px;
    align-items: center;
    gap: 8px;
    color: #a1a9b5;
    background: #f6f7f9;
    border: 1px solid #d9dee7;
    border-radius: 7px;
    cursor: not-allowed;
    font-size: 14px;

    svg { width: 18px; height: 18px; }
  }

  &__group-title {
    margin: 16px 0 8px;
    color: #5d6879;
    font-size: 13px;
    line-height: 20px;
  }

  &__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px 10px;

    &--more {
      padding-top: 8px;
      border-top: 1px solid #eef1f5;
    }
  }

  &__item {
    display: flex;
    min-width: 0;
    height: 44px;
    padding: 0 13px;
    align-items: center;
    gap: 9px;
    color: #253044;
    background: #fff;
    border: 1px solid #d6dce5;
    border-radius: 7px;
    cursor: pointer;
    font-size: 14px;
    text-align: left;
    transition: background-color 140ms ease, border-color 140ms ease, box-shadow 140ms ease;

    &:hover,
    &:focus-visible {
      background: #f7faff;
      border-color: #2f7cf6;
      box-shadow: 0 0 0 2px rgb(47 124 246 / 8%);
      outline: none;
    }

    svg {
      width: 19px;
      height: 19px;
      flex: 0 0 auto;
    }

    span {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  &__more {
    display: flex;
    height: 36px;
    margin-top: 8px;
    padding: 0 4px;
    align-items: center;
    gap: 9px;
    color: #354154;
    background: transparent;
    border: 0;
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;

    &:hover,
    &:focus-visible {
      color: #2f7cf6;
      background: #f5f8fd;
      outline: none;
    }

    svg { width: 18px; height: 18px; }
  }
}

@media (max-width: 600px) {
  .intelligent-insert-menu__grid { grid-template-columns: 1fr; }
}
</style>
