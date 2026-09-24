<script setup lang="ts">
import type {
  AssistantDraft,
  AssistantFormOption,
  AssistantTriggerType,
} from './assistant.types';
import {
  RiAddLine,
  RiArrowDownSLine,
  RiCloseCircleFill,
  RiCloseLine,
  RiFileList3Line,
  RiSearchLine,
  RiTimeLine,
  RiWebhookLine,
} from '@remixicon/vue';
import { computed, nextTick, reactive, shallowRef, useTemplateRef, watch } from 'vue';

defineOptions({ name: 'AssistantCreateDialog' });

const props = defineProps<{
  forms: AssistantFormOption[];
  defaultFormCode?: string;
  loadingForms?: boolean;
}>();

const emit = defineEmits<{
  create: [draft: AssistantDraft];
}>();
const visible = defineModel<boolean>({ default: false });

const form = reactive<AssistantDraft>({
  name: '未命名智能助手 Pro',
  triggerType: 'form',
  triggerFormCode: undefined,
  triggerFormName: undefined,
  tags: [],
});
const formPickerVisible = shallowRef(false);
const formKeyword = shallowRef('');
const moreTriggersVisible = shallowRef(false);
const tagEditorVisible = shallowRef(false);
const tagKeyword = shallowRef('');
const nameInput = useTemplateRef<HTMLInputElement>('nameInput');
const tagInput = useTemplateRef<HTMLInputElement>('tagInput');

const selectedForm = computed(() =>
  props.forms.find((item) => item.code === form.triggerFormCode),
);
const visibleForms = computed(() => {
  const keyword = formKeyword.value.trim().toLocaleLowerCase();
  if (!keyword) return props.forms;
  return props.forms.filter((item) => item.name.toLocaleLowerCase().includes(keyword));
});
const canSubmit = computed(
  () =>
    form.name.trim().length > 0 &&
    (form.triggerType !== 'form' || Boolean(form.triggerFormCode)),
);

watch(
  visible,
  async (isVisible) => {
    if (!isVisible) return;
    const defaultForm = props.forms.find((item) => item.code === props.defaultFormCode);
    Object.assign(form, {
      name: '未命名智能助手 Pro',
      triggerType: 'form',
      triggerFormCode: defaultForm?.code ?? props.forms[0]?.code,
      triggerFormName: defaultForm?.name ?? props.forms[0]?.name,
      tags: [],
    } satisfies AssistantDraft);
    formPickerVisible.value = false;
    formKeyword.value = '';
    moreTriggersVisible.value = false;
    tagEditorVisible.value = false;
    tagKeyword.value = '';
    await nextTick();
    nameInput.value?.focus();
    nameInput.value?.select();
  },
);

watch(
  () => props.forms,
  (forms) => {
    if (!visible.value || form.triggerType !== 'form') return;
    if (forms.some((item) => item.code === form.triggerFormCode)) return;
    const fallback = forms.find((item) => item.code === props.defaultFormCode) ?? forms[0];
    form.triggerFormCode = fallback?.code;
    form.triggerFormName = fallback?.name;
  },
);

function close(): void {
  visible.value = false;
}

function selectTrigger(triggerType: AssistantTriggerType): void {
  form.triggerType = triggerType;
  formPickerVisible.value = false;
}

function selectForm(option: AssistantFormOption): void {
  form.triggerFormCode = option.code;
  form.triggerFormName = option.name;
  formPickerVisible.value = false;
  formKeyword.value = '';
}

async function openTagEditor(): Promise<void> {
  tagEditorVisible.value = true;
  await nextTick();
  tagInput.value?.focus();
  // 标签编辑器位于可滚动弹窗底部，打开时确保完整候选区进入视口。
  tagInput.value?.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function addTag(): void {
  const tag = tagKeyword.value.trim();
  if (!tag || form.tags.includes(tag)) return;
  form.tags.push(tag);
  tagKeyword.value = '';
  tagEditorVisible.value = false;
}

function submit(): void {
  if (!canSubmit.value) return;
  emit('create', {
    name: form.name.trim(),
    triggerType: form.triggerType,
    triggerFormCode: form.triggerFormCode,
    triggerFormName: selectedForm.value?.name ?? form.triggerFormName,
    tags: [...form.tags],
  });
  close();
}
</script>

<template>
  <ElDialog
    v-model="visible"
    width="min(1200px, calc(100vw - 48px))"
    append-to-body
    destroy-on-close
    class="assistant-create-dialog"
    :close-on-click-modal="false"
  >
    <template #header>
      <h2 class="assistant-create-dialog__title">
        新建智能助手 Pro
      </h2>
    </template>

    <div class="assistant-create-dialog__body">
      <label class="assistant-create-dialog__field-label" for="assistant-name">名称</label>
      <input
        id="assistant-name"
        ref="nameInput"
        v-model="form.name"
        class="assistant-create-dialog__text-input"
        maxlength="80"
        autocomplete="off"
      >

      <h3 class="assistant-create-dialog__section-title">
        请选择触发方式
      </h3>
      <div class="assistant-create-dialog__trigger-list">
        <div
          role="radio"
          tabindex="0"
          class="assistant-create-dialog__trigger-card"
          :class="{ 'is-active': form.triggerType === 'form' }"
          :aria-checked="form.triggerType === 'form'"
          @click="selectTrigger('form')"
          @keydown.enter.prevent="selectTrigger('form')"
          @keydown.space.prevent="selectTrigger('form')"
        >
          <span class="assistant-create-dialog__trigger-icon is-form"><RiFileList3Line /></span>
          <span class="assistant-create-dialog__trigger-copy">
            <strong>表单触发</strong>
            <small>表单中数据发生变化时，触发自动化操作</small>
            <span v-if="form.triggerType === 'form'" class="assistant-create-dialog__picker-wrap">
              <button
                type="button"
                class="assistant-create-dialog__form-picker"
                :aria-expanded="formPickerVisible"
                @click.stop="formPickerVisible = !formPickerVisible"
              >
                <span>{{ selectedForm?.name ?? '请选择表单' }}</span>
                <RiCloseCircleFill
                  v-if="selectedForm && formPickerVisible"
                  class="assistant-create-dialog__picker-clear"
                  aria-hidden="true"
                />
                <RiArrowDownSLine v-else class="assistant-create-dialog__picker-arrow" aria-hidden="true" />
              </button>
              <div v-if="formPickerVisible" class="assistant-create-dialog__form-menu">
                <label class="assistant-create-dialog__search">
                  <RiSearchLine aria-hidden="true" />
                  <input v-model="formKeyword" placeholder="搜索" autocomplete="off">
                </label>
                <div class="assistant-create-dialog__form-options">
                  <button
                    v-for="option in visibleForms"
                    :key="option.code"
                    type="button"
                    :class="{ 'is-selected': option.code === form.triggerFormCode }"
                    @click.stop="selectForm(option)"
                  >
                    <span class="assistant-create-dialog__form-option-icon"><RiFileList3Line /></span>
                    <span>{{ option.name }}</span>
                  </button>
                  <p v-if="loadingForms" class="assistant-create-dialog__form-empty">正在加载表单…</p>
                  <p v-else-if="visibleForms.length === 0" class="assistant-create-dialog__form-empty">
                    未找到匹配的表单
                  </p>
                </div>
              </div>
            </span>
          </span>
        </div>

        <button
          type="button"
          class="assistant-create-dialog__trigger-card"
          :class="{ 'is-active': form.triggerType === 'schedule' }"
          @click="selectTrigger('schedule')"
        >
          <span class="assistant-create-dialog__trigger-icon is-schedule"><RiTimeLine /></span>
          <span class="assistant-create-dialog__trigger-copy">
            <strong>定时触发</strong>
            <small>根据设定的时间、频率周期性触发自动化操作</small>
          </span>
        </button>

        <button
          v-if="moreTriggersVisible"
          type="button"
          class="assistant-create-dialog__trigger-card"
          :class="{ 'is-active': form.triggerType === 'http' }"
          @click="selectTrigger('http')"
        >
          <span class="assistant-create-dialog__trigger-icon is-http"><RiWebhookLine /></span>
          <span class="assistant-create-dialog__trigger-copy">
            <strong>HTTP 触发</strong>
            <small>指定 URL 接收到第三方数据时，触发自动化操作</small>
          </span>
        </button>
      </div>

      <button
        v-if="!moreTriggersVisible"
        type="button"
        class="assistant-create-dialog__more-trigger"
        @click="moreTriggersVisible = true"
      >
        <span />更多触发方式<RiArrowDownSLine aria-hidden="true" /> <span />
      </button>

      <div class="assistant-create-dialog__tag-section">
        <h3 class="assistant-create-dialog__section-title">
          标签
        </h3>
        <div class="assistant-create-dialog__tags">
          <span v-for="tag in form.tags" :key="tag" class="assistant-create-dialog__tag">{{ tag }}</span>
          <button
            v-if="!tagEditorVisible"
            type="button"
            class="assistant-create-dialog__add-tag"
            @click="openTagEditor"
          >
            <RiAddLine />添加标签
          </button>
        </div>
        <div v-if="tagEditorVisible" class="assistant-create-dialog__tag-editor">
          <label>
            <RiSearchLine aria-hidden="true" />
            <input
              ref="tagInput"
              v-model="tagKeyword"
              maxlength="20"
              placeholder="输入标签名称"
              @keydown.enter.prevent="addTag"
            >
            <button type="button" aria-label="关闭标签输入" @click="tagEditorVisible = false">
              <RiCloseLine />
            </button>
          </label>
          <button type="button" :disabled="!tagKeyword.trim()" @click="addTag">
            <RiAddLine />创建标签“{{ tagKeyword.trim() || '新标签' }}”
          </button>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="assistant-create-dialog__footer">
        <ElButton size="large" @click="close">
          取消
        </ElButton>
        <ElButton size="large" type="primary" :disabled="!canSubmit" @click="submit">
          确定
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<style scoped lang="scss">
.assistant-create-dialog {
  &__title,
  &__section-title {
    margin: 0;
    color: #172033;
  }

  &__title {
    font-size: 24px;
    font-weight: 650;
    line-height: 32px;
  }

  &__body {
    height: min(1040px, calc(100vh - 240px));
    padding: 26px 32px 36px;
    overflow-x: hidden;
    overflow-y: auto;
    box-sizing: border-box;
  }

  &__field-label,
  &__section-title {
    display: block;
    font-size: 18px;
    font-weight: 650;
    line-height: 28px;
  }

  &__text-input {
    width: 100%;
    height: 44px;
    margin-top: 12px;
    padding: 0 14px;
    color: #172033;
    background: #fff;
    border: 1px solid #d7dce5;
    border-radius: 7px;
    box-sizing: border-box;
    font: inherit;
    outline: none;

    &:focus {
      border-color: #00afa2;
      box-shadow: 0 0 0 1px #00afa2;
    }
  }

  &__section-title {
    margin-top: 26px;
  }

  &__trigger-list {
    display: grid;
    margin-top: 14px;
    gap: 14px;
  }

  &__trigger-card {
    position: relative;
    display: flex;
    width: 100%;
    min-height: 112px;
    padding: 20px 22px;
    align-items: flex-start;
    gap: 18px;
    color: #172033;
    text-align: left;
    background: #fff;
    border: 1px solid #b8c0cd;
    border-radius: 5px;
    cursor: pointer;
    box-sizing: border-box;

    &:hover,
    &.is-active {
      border-color: #00afa2;
      box-shadow: inset 0 0 0 1px #00afa2;
    }
  }

  &__trigger-icon {
    display: inline-flex;
    width: 64px;
    height: 64px;
    flex: 0 0 64px;
    align-items: center;
    justify-content: center;
    border-radius: 50%;

    svg {
      width: 36px;
      height: 36px;
    }

    &.is-form {
      color: #2878f0;
      background: #eaf2ff;
    }

    &.is-schedule {
      color: #ef9f18;
      background: #fff5df;
    }

    &.is-http {
      color: #bd4cdd;
      background: #faecff;
    }
  }

  &__trigger-copy {
    display: flex;
    min-width: 0;
    flex: 1;
    flex-direction: column;

    strong {
      font-size: 18px;
      line-height: 26px;
    }

    small {
      margin-top: 4px;
      color: #596579;
      font-size: 16px;
      line-height: 24px;
    }
  }

  &__picker-wrap {
    position: relative;
    width: min(100%, 410px);
    margin-top: 8px;
  }

  &__form-picker {
    display: flex;
    width: 100%;
    height: 42px;
    padding: 0 14px;
    align-items: center;
    justify-content: space-between;
    color: #172033;
    background: #fff;
    border: 1px solid #d5dbe5;
    border-radius: 7px;
    cursor: pointer;
    font: inherit;

    &:hover,
    &[aria-expanded='true'] {
      border-color: #00afa2;
    }
  }

  &__picker-arrow {
    width: 20px;
    height: 20px;
    color: #3d4756;
  }

  &__picker-clear {
    width: 18px;
    height: 18px;
    color: #687384;
  }

  &__form-menu,
  &__tag-editor {
    position: absolute;
    z-index: 10;
    right: 0;
    left: 0;
    overflow: hidden;
    background: #fff;
    border: 1px solid #e3e7ed;
    border-radius: 8px;
    box-shadow: 0 12px 32px rgb(23 32 51 / 15%);
  }

  &__form-menu {
    top: 48px;
  }

  &__search,
  &__tag-editor label {
    display: flex;
    height: 42px;
    padding: 0 12px;
    align-items: center;
    gap: 8px;
    border-bottom: 1px solid #e6e9ef;

    svg {
      width: 20px;
      height: 20px;
      color: #39465a;
    }

    input {
      min-width: 0;
      flex: 1;
      border: 0;
      outline: 0;
      font: inherit;
    }
  }

  &__form-options {
    max-height: 250px;
    padding: 8px;
    overflow-y: auto;

    > button {
      display: flex;
      width: 100%;
      height: 42px;
      padding: 0 10px;
      align-items: center;
      gap: 10px;
      color: #172033;
      background: transparent;
      border: 0;
      border-radius: 7px;
      cursor: pointer;
      font: inherit;

      &:hover,
      &.is-selected {
        background: #e8f8f6;
      }
    }
  }

  &__form-option-icon {
    display: inline-flex;
    width: 24px;
    height: 24px;
    align-items: center;
    justify-content: center;
    color: #fff;
    background: #28a9eb;
    border-radius: 4px;

    svg {
      width: 17px;
      height: 17px;
    }
  }

  &__form-empty {
    margin: 18px 0;
    color: #8a94a3;
    text-align: center;
  }

  &__more-trigger {
    display: flex;
    width: 100%;
    margin-top: 14px;
    align-items: center;
    justify-content: center;
    gap: 20px;
    color: #263246;
    background: transparent;
    border: 0;
    cursor: pointer;
    font: inherit;

    span {
      height: 1px;
      flex: 1;
      background: #e0e4eb;
    }

    > svg {
      width: 18px;
      height: 18px;
      margin-left: -12px;
    }
  }

  &__tag-section {
    position: relative;
  }

  &__tags {
    display: flex;
    margin-top: 12px;
    flex-wrap: wrap;
    gap: 8px;
  }

  &__tag,
  &__add-tag {
    min-height: 32px;
    padding: 0 10px;
    display: inline-flex;
    align-items: center;
    color: #546074;
    border-radius: 5px;
    font-size: 15px;
  }

  &__tag {
    background: #eef0f4;
  }

  &__add-tag {
    gap: 4px;
    background: transparent;
    border: 1px dashed #cfd5df;
    cursor: pointer;

    svg {
      width: 18px;
      height: 18px;
    }
  }

  &__tag-editor {
    top: 72px;
    width: min(100%, 520px);

    label button {
      display: inline-flex;
      width: 30px;
      height: 30px;
      padding: 0;
      align-items: center;
      justify-content: center;
      color: #596579;
      background: transparent;
      border: 0;
      cursor: pointer;

      svg {
        width: 20px;
        height: 20px;
      }
    }

    > button {
      display: flex;
      width: calc(100% - 12px);
      height: 42px;
      margin: 6px;
      padding: 0 12px;
      align-items: center;
      gap: 8px;
      color: #172033;
      background: #f5f6f8;
      border: 0;
      border-radius: 6px;
      cursor: pointer;
      font: inherit;

      &:disabled {
        cursor: not-allowed;
        opacity: 0.55;
      }

      svg {
        width: 19px;
        height: 19px;
      }
    }
  }

  &__footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
}

:global(.assistant-create-dialog) {
  overflow: visible;
  top: auto;
  right: auto;
  left: auto;
  margin: 32px auto 0;
  color: #172033;
  color-scheme: light;
  background: #fff;
  border-radius: 14px;
}

:global(.assistant-create-dialog .el-dialog__header) {
  margin: 0;
  padding: 24px 32px 20px;
  border-bottom: 1px solid #e1e5eb;
}

:global(.assistant-create-dialog .el-dialog__headerbtn) {
  top: 18px;
  right: 22px;
}

:global(.assistant-create-dialog .el-dialog__close) {
  color: #303b4d;
  font-size: 25px;
}

:global(.assistant-create-dialog .el-dialog__body) {
  padding: 0;
}

:global(.assistant-create-dialog .el-dialog__footer) {
  padding: 16px 32px;
  border-top: 1px solid #e1e5eb;
}

:global(.assistant-create-dialog .el-button--primary) {
  color: #fff;
  background: #00afa2;
  border-color: #00afa2;
  --el-button-bg-color: #00afa2;
  --el-button-border-color: #00afa2;
  --el-button-hover-bg-color: #08baad;
  --el-button-hover-border-color: #08baad;
}

:global(.assistant-create-dialog .el-button:not(.el-button--primary)) {
  color: #172033;
  background: #fff;
  border-color: #d7dce5;
}

@media (max-width: 680px) {
  .assistant-create-dialog {
    &__body {
      min-height: 0;
      padding: 20px;
    }

    &__trigger-card {
      padding: 16px;
    }

    &__trigger-icon {
      width: 48px;
      height: 48px;
      flex-basis: 48px;

      svg {
        width: 28px;
        height: 28px;
      }
    }
  }
}
</style>
