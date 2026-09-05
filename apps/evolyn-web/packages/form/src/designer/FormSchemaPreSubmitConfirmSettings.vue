<script setup lang="ts">
import { RiAddLine, RiInformationLine, RiSettings3Line } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElPopover,
  ElSwitch,
  ElTooltip,
} from 'element-plus';
import type { FormItem } from '../schema/types';
import type { PreSubmitConfirmDraft } from './submit-validation-types';

const confirm = defineModel<PreSubmitConfirmDraft>({ required: true });
const props = defineProps<{ items: FormItem[] }>();
const dialogOpen = shallowRef(false);
const pickerOpen = shallowRef(false);

const variableItems = computed(() =>
  props.items.filter((item) => item.widget.type !== 'separator' && item.widget.type !== 'button'),
);
const enabled = computed({
  get: () => confirm.value.enable,
  set: (enable: boolean) => {
    confirm.value = { ...confirm.value, enable };
  },
});
const title = computed({
  get: () => confirm.value.title,
  set: (nextTitle: string) => {
    confirm.value = { ...confirm.value, title: nextTitle };
  },
});
const content = computed({
  get: () => confirm.value.content,
  set: (nextContent: string) => {
    confirm.value = { ...confirm.value, content: nextContent };
  },
});

function insertField(widgetName: string): void {
  content.value += `${content.value ? ' ' : ''}\${${widgetName}}`;
  pickerOpen.value = false;
}
</script>

<template>
  <section class="form-pre-submit-confirm" aria-label="提交时二次确认">
    <div class="form-pre-submit-confirm__heading">
      <span class="form-pre-submit-confirm__title">二次确认</span>
      <el-tooltip content="提交前提示填写人再次核对关键信息" placement="top">
        <el-icon class="form-pre-submit-confirm__help" aria-label="二次确认说明">
          <RiInformationLine />
        </el-icon>
      </el-tooltip>
    </div>
    <div class="form-pre-submit-confirm__control">
      <span class="form-pre-submit-confirm__caption">提交时弹出确认提示</span>
      <el-switch v-model="enabled" aria-label="开启提交二次确认" />
    </div>
    <button
      v-if="enabled"
      type="button"
      class="form-pre-submit-confirm__configure"
      @click="dialogOpen = true"
    >
      <span>设置提示文案</span>
      <el-icon><RiSettings3Line /></el-icon>
    </button>

    <el-dialog
      v-model="dialogOpen"
      append-to-body
      width="min(92vw, 640px)"
      class="form-pre-submit-confirm__dialog"
      title="提交二次确认"
    >
      <p class="form-pre-submit-confirm__intro">
        填写人确认后才会提交数据，取消时保留当前填写内容。
      </p>
      <el-form label-position="top" @submit.prevent>
        <el-form-item label="提示标题">
          <el-input v-model="title" :maxlength="100" placeholder="确认继续提交吗？" />
        </el-form-item>
        <el-form-item label="提示内容">
          <el-input
            v-model="content"
            type="textarea"
            :autosize="{ minRows: 4, maxRows: 8 }"
            :maxlength="1000"
            placeholder="请确认填写内容无误后继续提交。"
          />
        </el-form-item>
      </el-form>
      <el-popover
        v-model:visible="pickerOpen"
        placement="bottom-start"
        :width="330"
        trigger="click"
      >
        <template #reference>
          <el-button plain type="primary" :icon="RiAddLine">插入字段</el-button>
        </template>
        <div class="form-pre-submit-confirm__field-picker" role="listbox" aria-label="可插入字段">
          <button
            v-for="item in variableItems"
            :key="item.widget.widgetName"
            type="button"
            @click="insertField(item.widget.widgetName)"
          >
            <span>{{ item.label }}</span>
            <code>${{ '{' }}{{ item.widget.widgetName }}{{ '}' }}</code>
          </button>
          <p v-if="variableItems.length === 0">请先在画布中添加字段</p>
        </div>
      </el-popover>
      <template #footer>
        <el-button @click="dialogOpen = false">完成</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<style scoped lang="scss">
.form-pre-submit-confirm {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;

  &__heading,
  &__control {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  &__heading {
    justify-content: flex-start;
    gap: 6px;
  }
  &__title {
    font-size: 15px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  &__help {
    font-size: 16px;
    color: var(--el-text-color-secondary);
    cursor: help;
  }
  &__caption {
    font-size: 14px;
    color: var(--el-text-color-regular);
  }

  &__configure {
    display: inline-flex;
    gap: 6px;
    align-items: center;
    align-self: flex-start;
    padding: 0;
    font: inherit;
    font-size: 13px;
    color: var(--el-color-primary);
    cursor: pointer;
    background: none;
    border: 0;

    &:hover,
    &:focus-visible {
      color: var(--el-color-primary-light-3);
      outline: none;
    }
  }

  &__intro {
    margin: 0 0 20px;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 1.65;
  }

  &__field-picker {
    max-height: 270px;
    overflow-y: auto;
  }
  &__field-picker button {
    display: flex;
    width: 100%;
    min-height: 38px;
    align-items: center;
    justify-content: space-between;
    padding: 0 8px;
    color: var(--el-text-color-primary);
    text-align: left;
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 6px;
  }
  &__field-picker button:hover {
    background: var(--el-fill-color-light);
  }
  &__field-picker code {
    font-size: 11px;
    color: var(--el-color-primary);
  }
  &__field-picker p {
    margin: 8px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
}
</style>
