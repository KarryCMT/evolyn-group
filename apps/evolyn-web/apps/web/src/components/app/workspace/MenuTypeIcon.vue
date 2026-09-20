<script setup lang="ts">
import type { AppMenuType, FormType } from '~/types';
import { computed } from 'vue';
import { resolveMenuIcon, resolveMenuIconTone } from '../menuIcon';

defineOptions({ name: 'MenuTypeIcon' });

const props = withDefaults(
  defineProps<{
    /** folder 是前端树对后端 group 的展示别名。 */
    menuType: AppMenuType | 'folder';
    formType?: FormType | null;
    iconKey?: string | null;
    size?: number;
    label?: string;
  }>(),
  { formType: null, iconKey: null, size: 20, label: '菜单图标' },
);

const resolvedMenuType = computed<AppMenuType>(() =>
  props.menuType === 'folder' ? 'group' : props.menuType,
);
const icon = computed(() => resolveMenuIcon(resolvedMenuType.value, props.iconKey));
const tone = computed(() => resolveMenuIconTone(resolvedMenuType.value, props.formType));
</script>

<template>
  <span
    class="menu-type-icon"
    :class="`menu-type-icon--${tone}`"
    :style="{ '--menu-type-icon-size': `${props.size}px` }"
    role="img"
    :aria-label="props.label"
  >
    <component :is="icon" aria-hidden="true" />
  </span>
</template>

<style scoped lang="scss">
.menu-type-icon {
  display: inline-flex;
  width: var(--menu-type-icon-size);
  height: var(--menu-type-icon-size);
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;

  svg {
    width: 100%;
    height: 100%;
  }

  &--dashboard {
    color: #8a63f6;
  }

  &--form {
    color: #1aa7ee;
  }

  &--workflow {
    color: #ff9028;
  }

  &--group {
    color: #f5af2c;
  }

  &--page {
    color: #18a995;
  }
}
</style>
