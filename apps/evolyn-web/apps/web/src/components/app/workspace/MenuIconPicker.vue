<script setup lang="ts">
import type { AppMenuType, FormType } from '~/types';
import { computed, shallowRef } from 'vue';
import { defaultMenuIconKey, menuIconOptions } from '../menuIcon';
import MenuTypeIcon from './MenuTypeIcon.vue';

defineOptions({ name: 'MenuIconPicker' });

const props = withDefaults(
  defineProps<{
    /** folder 是前端树对后端 group 的展示别名。 */
    menuType: AppMenuType | 'folder';
    formType?: FormType | null;
    disabled?: boolean;
  }>(),
  { formType: null, disabled: false },
);
const iconKey = defineModel<string>({ required: true });
const popupVisible = shallowRef(false);
const resolvedMenuType = computed<AppMenuType>(() =>
  props.menuType === 'folder' ? 'group' : props.menuType,
);

const displayIconKey = computed(() =>
  menuIconOptions.some((option) => option.key === iconKey.value)
    ? iconKey.value
    : defaultMenuIconKey(resolvedMenuType.value, props.formType),
);

function selectIcon(key: string) {
  if (props.disabled) return;
  iconKey.value = key;
  popupVisible.value = false;
}
</script>

<template>
  <el-popover
    v-model:visible="popupVisible"
    placement="bottom-start"
    trigger="click"
    :width="286"
    popper-class="menu-icon-picker__popper"
    :disabled="props.disabled"
  >
    <template #reference>
      <button
        class="menu-icon-picker__trigger"
        type="button"
        :disabled="props.disabled"
        aria-label="选择菜单图标"
      >
        <MenuTypeIcon
          :menu-type="props.menuType"
          :form-type="props.formType"
          :icon-key="displayIconKey"
          :size="25"
          label="当前菜单图标"
        />
      </button>
    </template>

    <div class="menu-icon-picker__grid" role="listbox" aria-label="菜单图标">
      <button
        v-for="option in menuIconOptions"
        :key="option.key"
        class="menu-icon-picker__option"
        :class="{ 'menu-icon-picker__option--selected': displayIconKey === option.key }"
        type="button"
        role="option"
        :aria-label="`选择${option.label}图标`"
        :aria-selected="displayIconKey === option.key"
        @click="selectIcon(option.key)"
      >
        <MenuTypeIcon
          :menu-type="props.menuType"
          :form-type="props.formType"
          :icon-key="option.key"
          :size="22"
          :label="option.label"
        />
      </button>
    </div>
  </el-popover>
</template>

<style scoped lang="scss">
.menu-icon-picker__trigger {
  display: inline-flex;
  width: 56px;
  height: 56px;
  padding: 0;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-medium);
  transition:
    border-color 0.16s ease,
    background-color 0.16s ease;

  &:hover:not(:disabled) {
    background: var(--el-fill-color-lighter);
    border-color: var(--el-color-primary-light-5);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }

  &:disabled {
    cursor: not-allowed;
  }
}

.menu-icon-picker__grid {
  display: grid;
  grid-template-columns: repeat(5, 42px);
  gap: var(--el-space-sm);
}

.menu-icon-picker__option {
  display: inline-flex;
  width: 42px;
  height: 42px;
  padding: 0;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-medium);
  transition:
    border-color 0.16s ease,
    background-color 0.16s ease,
    transform 0.16s ease;

  &:hover {
    background: var(--el-fill-color-light);
    border-color: var(--el-color-primary-light-5);
    transform: translateY(-1px);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }

  &--selected {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-5);
  }
}
</style>

<style lang="scss">
.menu-icon-picker__popper.el-popper {
  padding: var(--el-space-md);
  border-color: var(--el-border-color-lighter);
  box-shadow: var(--el-box-shadow-light);
}
</style>
