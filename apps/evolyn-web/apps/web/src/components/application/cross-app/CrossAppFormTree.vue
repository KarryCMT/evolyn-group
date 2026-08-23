<script setup lang="ts">
import type { CrossAppForm, CrossAppFormGroup } from './crossApp.types';
import { RiFileList3Fill, RiFolder3Fill, RiGitBranchFill } from '@remixicon/vue';
import { computed } from 'vue';

defineOptions({ name: 'CrossAppFormTree' });

const props = defineProps<{
  groups: CrossAppFormGroup[];
  selectedFormIds: string[];
}>();

const emit = defineEmits<{
  toggleForm: [id: string];
  toggleGroup: [formIds: string[]];
}>();

const selectedIdSet = computed(() => new Set(props.selectedFormIds));

function formIds(group: CrossAppFormGroup) {
  return group.forms.map((form) => form.id);
}

function isFormSelected(form: CrossAppForm) {
  return selectedIdSet.value.has(form.id);
}

function isGroupSelected(group: CrossAppFormGroup) {
  const ids = formIds(group);
  return ids.length > 0 && ids.every((id) => selectedIdSet.value.has(id));
}

function isGroupIndeterminate(group: CrossAppFormGroup) {
  const ids = formIds(group);
  const selectedCount = ids.filter((id) => selectedIdSet.value.has(id)).length;
  return selectedCount > 0 && selectedCount < ids.length;
}
</script>

<template>
  <!-- 表单树与应用列表同样由 Element Plus 提供滚动行为。 -->
  <el-scrollbar class="cross-app-form-tree" always>
    <div class="cross-app-form-tree__content">
      <section v-for="group in props.groups" :key="group.id" class="cross-app-form-tree__group">
        <div
          class="cross-app-form-tree__row cross-app-form-tree__row--group"
          role="checkbox"
          tabindex="0"
          :aria-checked="isGroupSelected(group)"
          @click="emit('toggleGroup', formIds(group))"
          @keydown.enter.prevent="emit('toggleGroup', formIds(group))"
          @keydown.space.prevent="emit('toggleGroup', formIds(group))"
        >
          <el-checkbox
            :model-value="isGroupSelected(group)"
            :indeterminate="isGroupIndeterminate(group)"
            @click.stop
            @change="emit('toggleGroup', formIds(group))"
          />
          <RiFolder3Fill
            class="cross-app-form-tree__icon cross-app-form-tree__icon--folder"
            aria-hidden="true"
          />
          <span class="cross-app-form-tree__name">{{ group.name }}</span>
        </div>
        <div
          v-for="form in group.forms"
          :key="form.id"
          class="cross-app-form-tree__row cross-app-form-tree__row--form"
          role="checkbox"
          tabindex="0"
          :aria-checked="isFormSelected(form)"
          @click="emit('toggleForm', form.id)"
          @keydown.enter.prevent="emit('toggleForm', form.id)"
          @keydown.space.prevent="emit('toggleForm', form.id)"
        >
          <el-checkbox
            :model-value="isFormSelected(form)"
            @click.stop
            @change="emit('toggleForm', form.id)"
          />
          <component
            :is="form.kind === 'workflow-form' ? RiGitBranchFill : RiFileList3Fill"
            class="cross-app-form-tree__icon"
            :class="{
              'cross-app-form-tree__icon--workflow': form.kind === 'workflow-form',
              'cross-app-form-tree__icon--form': form.kind === 'form',
            }"
            aria-hidden="true"
          />
          <span class="cross-app-form-tree__name">{{ form.name }}</span>
        </div>
      </section>
    </div>
  </el-scrollbar>
</template>

<style scoped lang="scss">
.cross-app-form-tree {
  height: 0;
  min-height: 0;
  flex: 1;

  &__content {
    box-sizing: border-box;
    min-height: 100%;
    padding: 10px 20px 28px;
  }

  &__group {
    display: flex;
    flex-direction: column;
  }

  &__row {
    display: flex;
    min-height: 42px;
    padding: 0 8px;
    border-radius: 6px;
    align-items: center;
    gap: 8px;
    color: var(--el-text-color-regular);
    cursor: pointer;
    transition: background-color 0.16s ease;

    &:hover {
      background: var(--el-fill-color-light);
    }
  }

  &__row--group {
    margin-top: 4px;
    color: var(--el-text-color-primary);
    font-weight: 600;
  }

  &__row--form {
    padding-left: 36px;
  }

  &__icon {
    width: 19px;
    height: 19px;
    flex: 0 0 auto;
  }

  &__icon--folder {
    color: var(--el-color-warning);
  }

  &__icon--form {
    color: var(--el-color-primary);
  }

  &__icon--workflow {
    color: var(--el-color-success);
  }

  &__name {
    overflow: hidden;
    font-size: 14px;
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :deep(.el-checkbox) {
    height: auto;
    margin-right: 0;
  }
}
</style>
