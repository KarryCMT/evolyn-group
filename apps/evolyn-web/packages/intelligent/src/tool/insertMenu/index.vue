<script setup lang="ts">
import { computed, shallowRef } from 'vue';
import type { IntelligentActionType, IntelligentPosition } from '../../schema';
import { intelligentNodeTemplates } from '../../mock/nodeTemplates';
import AddIcon from '../../icon/addIcon.vue';
import CommonIcon from '../../icon/commonIcon.vue';
import FilterIcon from '../../icon/filterIcon.vue';
import ReactionIcon from '../../icon/reactionIcon.vue';

defineOptions({ name: 'IntelligentInsertMenu' });

const props = defineProps<{
  position: IntelligentPosition;
}>();
const emit = defineEmits<{
  select: [actionType: IntelligentActionType];
}>();
const visible = defineModel<boolean>({ default: false });

const keyword = shallowRef('');
const menuStyle = computed(() => ({ left: `${props.position.x}px`, top: `${props.position.y}px` }));
const filteredTemplates = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  if (!normalized) return intelligentNodeTemplates;
  return intelligentNodeTemplates.filter((item) =>
    `${item.name}${item.description}`.toLocaleLowerCase().includes(normalized),
  );
});

function select(actionType: IntelligentActionType): void {
  emit('select', actionType);
  visible.value = false;
  keyword.value = '';
}
</script>

<template>
  <div v-if="visible" class="intelligent-insert-menu" :style="menuStyle" role="dialog" aria-label="添加执行节点">
    <label class="intelligent-insert-menu__search">
      <FilterIcon size="16" />
      <input v-model="keyword" placeholder="搜索节点" autocomplete="off">
    </label>
    <div class="intelligent-insert-menu__list">
      <button
        v-for="item in filteredTemplates"
        :key="item.type"
        type="button"
        @click="select(item.type)"
      >
        <span class="intelligent-insert-menu__icon">
          <AddIcon v-if="item.group === 'data'" size="17" />
          <ReactionIcon v-else-if="item.group === 'message'" size="17" />
          <CommonIcon v-else size="17" />
        </span>
        <span>
          <strong>{{ item.name }}</strong>
          <small>{{ item.description }}</small>
        </span>
      </button>
      <p v-if="filteredTemplates.length === 0">未找到匹配节点</p>
    </div>
  </div>
</template>

<style scoped lang="scss">
.intelligent-insert-menu {
  position: absolute;
  z-index: 12;
  width: 292px;
  padding: 10px;
  background: #fff;
  border: 1px solid #e0e5ec;
  border-radius: 10px;
  box-shadow: 0 12px 36px rgb(31 43 61 / 18%);
  transform: translate(-50%, 10px);

  &__search {
    display: flex;
    height: 36px;
    padding: 0 10px;
    align-items: center;
    gap: 8px;
    color: #7d8795;
    background: #f5f7fa;
    border-radius: 6px;

    input { min-width: 0; flex: 1; background: transparent; border: 0; outline: 0; }
  }

  &__list {
    display: grid;
    max-height: 336px;
    margin-top: 8px;
    overflow-y: auto;
    gap: 3px;

    button {
      display: flex;
      width: 100%;
      padding: 9px 8px;
      align-items: center;
      gap: 10px;
      color: #172033;
      background: transparent;
      border: 0;
      border-radius: 7px;
      cursor: pointer;
      text-align: left;

      &:hover { background: #edf9f7; }
      > span:last-child { display: grid; min-width: 0; gap: 3px; }
      strong { font-size: 14px; font-weight: 600; }
      small { overflow: hidden; color: #7a8596; text-overflow: ellipsis; white-space: nowrap; }
    }

    p { margin: 18px 0; color: #929baa; text-align: center; }
  }

  &__icon {
    display: inline-flex;
    width: 30px;
    height: 30px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    color: #00a99d;
    background: #e9f8f6;
    border-radius: 7px;
  }
}
</style>
