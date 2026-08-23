<script setup lang="ts">
import { RiRecycleFill } from '@remixicon/vue';
import { ElScrollbar } from 'element-plus';
import { formFieldGroups, type FormFieldPreset } from '../schema';
import { formFieldIcons } from './fieldIcons';

const emit = defineEmits<{
  selectField: [preset: FormFieldPreset];
  openRecycleBin: [];
}>();
</script>

<template>
  <aside class="form-field-palette" aria-label="字段组件">
    <ElScrollbar class="form-field-palette__scrollbar">
      <section v-for="group in formFieldGroups" :key="group.key" class="form-field-palette__group">
        <div class="form-field-palette__heading">
          <h2 class="form-field-palette__title">{{ group.title }}</h2>
          <span v-if="group.key === 'common'" class="form-field-palette__ai-tag">AI 推荐字段</span>
        </div>
        <div class="form-field-palette__grid">
          <button
            v-for="field in group.fields"
            :key="field.type"
            class="form-field-palette__item"
            type="button"
            @click="emit('selectField', field)"
          >
            <span class="form-field-palette__symbol" aria-hidden="true">
              <component :is="formFieldIcons[field.type]" />
            </span>
            <span class="form-field-palette__name">{{ field.title }}</span>
          </button>
        </div>
      </section>
    </ElScrollbar>
    <button class="form-field-palette__recycle" type="button" @click="emit('openRecycleBin')">
      <RiRecycleFill />
      <span>字段回收站</span>
    </button>
  </aside>
</template>

<style scoped lang="scss">
.form-field-palette {
  display: flex;
  box-sizing: border-box;
  width: 260px;
  min-width: 260px;
  max-width: 260px;
  overflow: hidden;
  flex-direction: column;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-lighter);

  &__scrollbar {
    min-height: 0;
    flex: 1;
  }

  &__group {
    margin: 20px 14px 26px 15px;
  }

  &__heading,
  &__item,
  &__recycle {
    display: flex;
    align-items: center;
  }

  &__heading {
    min-height: 28px;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 10px;
  }

  &__title {
    margin: 0;
    font-size: 16px;
    font-weight: 650;
    line-height: 24px;
  }

  &__ai-tag {
    padding: 3px 8px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: var(--el-border-radius-small);
    font-size: 12px;
    font-weight: 600;
    line-height: 18px;
  }

  &__grid {
    display: grid;
    grid-template-columns: repeat(2, 110px);
    gap: 10px;
  }

  &__item {
    box-sizing: border-box;
    width: 110px;
    min-width: 110px;
    max-width: 110px;
    height: 32px;
    padding: 0 10px;
    gap: 8px;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);
    text-align: left;
    transition:
      color 0.18s ease,
      background-color 0.18s ease,
      border-color 0.18s ease,
      transform 0.18s ease;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
      border-color: var(--el-color-primary-light-5);
      transform: translateY(-1px);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__symbol {
    display: inline-flex;
    width: 18px;
    height: 18px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    border-radius: 4px;
    font-size: 11px;
    font-weight: 700;

    svg {
      width: 14px;
      height: 14px;
    }
  }

  &__name {
    overflow: hidden;
    font-size: 12px;
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__recycle {
    min-height: 50px;
    justify-content: center;
    gap: 7px;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: var(--el-bg-color);
    border: 0;
    border-top: 1px solid var(--el-border-color-lighter);
    font-size: 14px;
    font-weight: 600;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }
  }
}
</style>
