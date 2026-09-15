<script setup lang="ts">
import type { ApplicationIcon, ApplicationItem, UpdateApplicationPayload } from '~/types';
import { shallowRef, watch } from 'vue';
import ApplicationIconPicker from '~/components/application/ApplicationIconPicker.vue';

defineOptions({ name: 'ApplicationBasicSettingsPanel' });

const props = defineProps<{
  application: ApplicationItem;
  saving?: boolean;
}>();

const emit = defineEmits<{
  update: [payload: UpdateApplicationPayload];
  copyId: [value: string];
  configureHome: [];
  configureUrl: [];
}>();

const nameEditing = shallowRef(false);
const nameDraft = shallowRef(props.application.name);
// 当前接口只支持名称和图标持久化；两个开关保留为可交互的界面预览状态。
const watermarkEnabled = shallowRef(false);
const attachmentRestricted = shallowRef(false);

watch(
  () => props.application.name,
  (name) => {
    if (!nameEditing.value) nameDraft.value = name;
  },
);

function startNameEdit() {
  nameDraft.value = props.application.name;
  nameEditing.value = true;
}

function cancelNameEdit() {
  nameDraft.value = props.application.name;
  nameEditing.value = false;
}

function submitNameEdit() {
  const name = nameDraft.value.trim();
  if (!name || name === props.application.name) {
    cancelNameEdit();
    return;
  }
  nameEditing.value = false;
  emit('update', { name });
}

/** 图标选择器通过受控值回传；仅在图标确有变化时请求持久化。 */
function updateIcon(icon: ApplicationIcon) {
  const current = props.application.icon;
  const unchanged =
    current.type === 'custom' && icon.type === 'custom'
      ? current.name === icon.name
      : current.type === 'remix' && icon.type === 'remix'
        ? current.name === icon.name && current.background === icon.background
        : false;
  if (!unchanged) emit('update', { icon });
}
</script>

<template>
  <article class="application-basic-settings" aria-label="应用设置">
    <header class="application-basic-settings__header">
      <h1 class="application-basic-settings__title">应用设置</h1>
    </header>

    <!-- 标题独立于滚动容器，只有基础设置表格在内容超出时滚动。 -->
    <el-scrollbar class="application-basic-settings__scrollbar">
      <div class="application-basic-settings__body">
        <section class="application-basic-settings__table" aria-label="应用基础信息">
          <div class="application-basic-settings__row">
            <div class="application-basic-settings__label">应用名称</div>
            <div class="application-basic-settings__value application-basic-settings__value--name">
              <template v-if="nameEditing">
                <el-input
                  v-model="nameDraft"
                  class="application-basic-settings__name-input"
                  maxlength="64"
                  autofocus
                  @keyup.enter="submitNameEdit"
                  @keyup.esc="cancelNameEdit"
                />
                <el-button
                  type="primary"
                  size="small"
                  :loading="props.saving"
                  @click="submitNameEdit"
                >
                  保存
                </el-button>
                <button
                  class="application-basic-settings__text-button"
                  type="button"
                  :disabled="props.saving"
                  @click="cancelNameEdit"
                >
                  取消
                </button>
              </template>
              <template v-else>
                <span class="application-basic-settings__name">{{ props.application.name }}</span>
                <button
                  class="application-basic-settings__text-button"
                  type="button"
                  @click="startNameEdit"
                >
                  修改
                </button>
              </template>
            </div>
          </div>

          <div class="application-basic-settings__row">
            <div class="application-basic-settings__label">应用图标</div>
            <div class="application-basic-settings__value application-basic-settings__value--icon">
              <ApplicationIconPicker
                :model-value="props.application.icon"
                @update:model-value="updateIcon"
              />
            </div>
          </div>

          <div class="application-basic-settings__row">
            <div class="application-basic-settings__label">应用首页</div>
            <div class="application-basic-settings__value">
              <el-button plain type="primary" @click="emit('configureHome')"> 设置 </el-button>
            </div>
          </div>

          <div class="application-basic-settings__row">
            <div class="application-basic-settings__label">应用URL</div>
            <div
              class="application-basic-settings__value application-basic-settings__value--inline"
            >
              <el-button plain type="primary" @click="emit('configureUrl')"> 设置 </el-button>
              <span class="application-basic-settings__hint"
                >自定义应用URL，可使用URL直接打开该应用</span
              >
            </div>
          </div>

          <div class="application-basic-settings__row">
            <div class="application-basic-settings__label">应用ID</div>
            <div
              class="application-basic-settings__value application-basic-settings__value--inline"
            >
              <span class="application-basic-settings__id">{{ props.application.code }}</span>
              <button
                class="application-basic-settings__text-button"
                type="button"
                @click="emit('copyId', props.application.code)"
              >
                复制
              </button>
            </div>
          </div>

          <div class="application-basic-settings__row">
            <div class="application-basic-settings__label">应用水印</div>
            <div class="application-basic-settings__value">
              <el-switch v-model="watermarkEnabled" aria-label="应用水印" />
            </div>
          </div>

          <div class="application-basic-settings__row">
            <div class="application-basic-settings__label">附件管控</div>
            <div
              class="application-basic-settings__value application-basic-settings__value--inline"
            >
              <el-switch v-model="attachmentRestricted" aria-label="附件管控" />
              <span class="application-basic-settings__hint"
                >开启后，成员在表单和仪表盘中无法下载、打印和导出附件</span
              >
            </div>
          </div>
        </section>
      </div>
    </el-scrollbar>
  </article>
</template>

<style scoped lang="scss">
.application-basic-settings {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  background: var(--el-bg-color);

  &__header {
    display: flex;
    height: 82px;
    padding: 0 var(--el-space-3xl);
    align-items: center;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__title {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    line-height: 28px;
  }

  &__body {
    padding: var(--el-space-3xl) var(--el-space-3xl) 72px;
  }

  &__scrollbar {
    min-height: 0;
    flex: 1;
  }

  &__table {
    overflow: visible;
    border-top: 1px solid var(--el-border-color-lighter);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__row {
    display: grid;
    min-height: 76px;
    grid-template-columns: 170px minmax(0, 1fr);
    border-bottom: 1px solid var(--el-border-color-lighter);

    &:last-child {
      border-bottom: 0;
    }
  }

  &__label {
    box-sizing: border-box;
    display: flex;
    padding: var(--el-space-lg) var(--el-space-3xl);
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-lighter);
    font-size: var(--el-font-size-medium);
    text-align: center;
  }

  &__value {
    position: relative;
    box-sizing: border-box;
    display: flex;
    min-width: 0;
    padding: var(--el-space-lg) var(--el-space-4xl);
    align-items: center;
    gap: var(--el-space-2xl);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
  }

  &__value--name {
    gap: var(--el-space-xl);
  }

  &__value--icon {
    min-height: 76px;
  }

  &__value--inline {
    gap: var(--el-space-3xl);
  }

  &__name,
  &__id {
    color: var(--el-text-color-primary);
  }

  &__name-input {
    width: min(360px, 54vw);
  }

  &__text-button {
    padding: var(--el-space-xs) var(--el-space-xs);
    border: 0;
    border-radius: var(--el-border-radius-base);
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    font: inherit;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    &:hover:not(:disabled) {
      color: var(--el-color-primary-dark-2);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }

    &:disabled {
      cursor: not-allowed;
      color: var(--el-text-color-placeholder);
    }
  }

  &__hint {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
  }
}

@media (max-width: 760px) {
  .application-basic-settings {
    &__header {
      height: 64px;
      padding: 0 var(--el-space-2xl);
    }

    &__body {
      padding: var(--el-space-xl) var(--el-space-2xl) var(--el-space-5xl);
    }

    &__row {
      grid-template-columns: 112px minmax(0, 1fr);
    }

    &__label,
    &__value {
      padding-right: var(--el-space-2xl);
      padding-left: var(--el-space-2xl);
    }

    &__value {
      flex-wrap: wrap;
    }
  }
}
</style>
