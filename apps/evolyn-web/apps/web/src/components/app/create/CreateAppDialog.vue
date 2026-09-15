<script setup lang="ts">
import {
  RiArrowRightSLine,
  RiCloseFill,
  RiLightbulbFlashFill,
  RiRefreshFill,
  RiThumbUpFill,
} from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import customAppBackground from '~/assets/images/customer_bg.png';
import templateCenterBackground from '~/assets/images/template-center-banner_bg.png';
import AppStarterCard from './AppStarterCard.vue';
import AppTemplateCard from './AppTemplateCard.vue';
import BlankAppDialog, { type BlankAppDraft } from './BlankAppDialog.vue';
import {
  appStarters,
  appTemplateBatches,
  type AppStarter,
  type AppTemplate,
} from './appTemplateCatalog';

defineOptions({ name: 'CreateAppDialog' });

defineProps<{
  /** 空白应用异步创建处理：resolve true 关闭两级弹窗，false 保持开启（失败保留填写内容） */
  submitBlank: (draft: BlankAppDraft) => Promise<boolean>;
}>();

const emit = defineEmits<{
  selectStarter: [starter: Exclude<AppStarter, { id: 'blank' }>];
  selectTemplate: [template: AppTemplate];
  openTemplateCenter: [];
  requestCustomization: [];
}>();

const visible = defineModel<boolean>({ default: false });
const activeTemplateBatch = shallowRef(0);
const blankAppVisible = shallowRef(false);
const visibleTemplates = computed(() => appTemplateBatches[activeTemplateBatch.value] ?? []);

function selectStarter(starter: AppStarter) {
  if (starter.id === 'blank') {
    blankAppVisible.value = true;
    return;
  }

  emit('selectStarter', starter);
}

// 关闭一级弹窗时一并收起二级表单，避免传送到 body 的子弹窗失去上下文。
// 提交进行中由二级表单自身禁用关闭入口，此处不强制打断请求。
watch(visible, (isVisible) => {
  if (!isVisible) blankAppVisible.value = false;
});

// 二级表单异步提交成功后才关闭两级弹窗；失败由处理方提示并保留表单
function handleBlankSuccess() {
  blankAppVisible.value = false;
  visible.value = false;
}

function showNextTemplates() {
  activeTemplateBatch.value = (activeTemplateBatch.value + 1) % appTemplateBatches.length;
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="create-app-dialog"
    width="880px"
    top="8vh"
    :show-close="false"
    :close-on-click-modal="false"
    append-to-body
  >
    <template #header>
      <header class="create-app-dialog__header">
        <h2 class="create-app-dialog__heading">新建应用</h2>
        <button
          class="create-app-dialog__close"
          type="button"
          aria-label="关闭新建应用"
          @click="visible = false"
        >
          <RiCloseFill />
        </button>
      </header>
    </template>

    <div class="create-app-dialog__body">
      <section class="create-app-dialog__section" aria-labelledby="app-starter-heading">
        <div class="create-app-dialog__section-heading">
          <RiLightbulbFlashFill aria-hidden="true" />
          <h3 id="app-starter-heading">猜你想要</h3>
        </div>
        <div class="create-app-dialog__starter-grid">
          <AppStarterCard
            v-for="starter in appStarters"
            :key="starter.id"
            :starter="starter"
            @select="selectStarter"
          />
        </div>
      </section>

      <section class="create-app-dialog__section" aria-labelledby="app-template-heading">
        <div class="create-app-dialog__section-heading create-app-dialog__section-heading--split">
          <span class="create-app-dialog__section-title">
            <RiThumbUpFill aria-hidden="true" />
            <h3 id="app-template-heading">热门模板</h3>
          </span>
          <button class="create-app-dialog__refresh" type="button" @click="showNextTemplates">
            <RiRefreshFill aria-hidden="true" />
            换一批
          </button>
        </div>
        <div class="create-app-dialog__template-grid">
          <AppTemplateCard
            v-for="template in visibleTemplates"
            :key="template.id"
            :template="template"
            @select="emit('selectTemplate', $event)"
          />
        </div>
      </section>

      <section class="create-app-dialog__section" aria-labelledby="app-more-heading">
        <div class="create-app-dialog__section-heading">
          <RiLightbulbFlashFill aria-hidden="true" />
          <h3 id="app-more-heading">没有找到想要的模板？试试下面</h3>
        </div>
        <div class="create-app-dialog__more-grid">
          <button
            class="create-app-dialog__promotion create-app-dialog__promotion--templates"
            type="button"
            :style="{ backgroundImage: `url(${templateCenterBackground})` }"
            @click="emit('openTemplateCenter')"
          >
            <span class="create-app-dialog__promotion-content">
              <strong>模板中心</strong>
              <span>海量优质模板，总有一款适合你</span>
            </span>
            <RiArrowRightSLine aria-hidden="true" />
          </button>
          <button
            class="create-app-dialog__promotion create-app-dialog__promotion--custom"
            type="button"
            :style="{ backgroundImage: `url(${customAppBackground})` }"
            @click="emit('requestCustomization')"
          >
            <span class="create-app-dialog__promotion-content">
              <strong>定制应用</strong>
              <span>官方团队为你个性化定制</span>
            </span>
            <RiArrowRightSLine aria-hidden="true" />
          </button>
        </div>
      </section>
    </div>
  </el-dialog>
  <BlankAppDialog v-model="blankAppVisible" :submit="submitBlank" @success="handleBlankSuccess" />
</template>

<!-- 弹窗会传送至 body，样式须通过唯一块类限定。 -->
<style lang="scss">
.create-app-dialog.el-dialog {
  display: flex;
  max-width: calc(100vw - 32px);
  height: 665px;
  max-height: calc(100vh - 32px);
  margin-bottom: 0;
  overflow: hidden;
  flex-direction: column;
  border-radius: var(--el-border-radius-round);
}

.create-app-dialog .el-dialog__header {
  flex: 0 0 auto;
  padding: 0;
  margin: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.create-app-dialog .el-dialog__body {
  flex: 1;
  min-height: 0;
  padding: 0;
  overflow: hidden;
}

.create-app-dialog__header {
  display: flex;
  height: 56px;
  padding: 0 var(--el-space-3xl) 0 var(--el-space-5xl);
  align-items: center;
  justify-content: space-between;
}

.create-app-dialog__heading {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-medium);
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.15;
}

.create-app-dialog__close,
.create-app-dialog__refresh {
  display: inline-flex;
  padding: 0;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  background: transparent;
  border: 0;
}

.create-app-dialog__close {
  width: 32px;
  height: 32px;
  color: var(--el-text-color-regular);
  border-radius: var(--el-border-radius-base);
  font-size: var(--el-font-size-medium);
  transition:
    color 0.2s ease,
    background-color 0.2s ease;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }
}

.create-app-dialog__body {
  box-sizing: border-box;
  display: flex;
  height: 100%;
  min-height: 0;
  padding: var(--el-space-xl) var(--el-space-3xl);
  flex-direction: column;
  gap: var(--el-space-xl);
}

.create-app-dialog__section {
  display: flex;
  flex-direction: column;
  gap: var(--el-space-md);
}

.create-app-dialog__section-heading,
.create-app-dialog__section-title {
  display: inline-flex;
  align-items: center;
  gap: var(--el-space-md);
  color: var(--el-text-color-primary);

  > svg {
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-medium);
  }

  h3 {
    margin: 0;
    font-size: var(--el-font-size-medium);
    font-weight: 650;
    letter-spacing: -0.01em;
    line-height: 1.3;
  }

  &--split {
    justify-content: space-between;
  }
}

.create-app-dialog__refresh {
  height: 32px;
  padding: 0 var(--el-space-xs);
  gap: var(--el-space-xs);
  color: var(--el-color-primary);
  border-radius: var(--el-border-radius-base);
  font-size: var(--el-font-size-base);
  transition: background-color 0.2s ease;

  svg {
    font-size: var(--el-font-size-medium);
  }

  &:hover {
    background: var(--el-color-primary-light-9);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }
}

.create-app-dialog__starter-grid,
.create-app-dialog__template-grid,
.create-app-dialog__more-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--el-space-lg);
}

.create-app-dialog__promotion {
  display: flex;
  min-height: 108px;
  padding: var(--el-space-xl) var(--el-space-2xl);
  align-items: flex-start;
  justify-content: space-between;
  color: var(--el-color-white);
  text-align: left;
  cursor: pointer;
  background-color: var(--el-color-primary);
  background-position: center;
  background-repeat: no-repeat;
  background-size: cover;
  border: 0;
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);
  transition:
    box-shadow 0.2s ease,
    transform 0.2s ease;

  &:hover {
    box-shadow: var(--el-box-shadow-light);
    transform: translateY(-2px);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 3px;
  }

  &--templates {
    grid-column: span 3;
  }

  &--custom {
    grid-column: span 1;
  }

  svg {
    margin-top: var(--el-space-xs);
    font-size: 28px;
  }
}

.create-app-dialog__promotion-content {
  display: flex;
  max-width: 340px;
  flex-direction: column;
  gap: var(--el-space-sm);

  strong {
    font-size: var(--el-font-size-large);
    font-weight: 700;
    line-height: 1.2;
  }

  span {
    font-size: var(--el-font-size-small);
    line-height: 1.5;
  }
}

@media (max-width: 1100px) {
  .create-app-dialog__starter-grid,
  .create-app-dialog__template-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .create-app-dialog__more-grid {
    grid-template-columns: 1fr;
  }

  .create-app-dialog__promotion--templates,
  .create-app-dialog__promotion--custom {
    grid-column: auto;
  }
}

@media (max-width: 720px) {
  .create-app-dialog.el-dialog {
    width: 100vw !important;
    height: 100vh;
    border-radius: 0;
  }

  .create-app-dialog__header {
    height: 52px;
    padding: 0 var(--el-space-lg) 0 var(--el-space-2xl);
  }

  .create-app-dialog__heading {
    font-size: var(--el-font-size-large);
    line-height: 26px;
  }

  .create-app-dialog__body {
    padding: var(--el-space-3xl) var(--el-space-xl) var(--el-space-4xl);
    gap: var(--el-space-4xl);
  }

  .create-app-dialog__section {
    gap: var(--el-space-xl);
  }

  .create-app-dialog__section-heading,
  .create-app-dialog__section-title {
    gap: var(--el-space-md);

    > svg {
      font-size: var(--el-font-size-medium);
    }

    h3 {
      font-size: var(--el-font-size-large);
    }
  }

  .create-app-dialog__starter-grid,
  .create-app-dialog__template-grid {
    grid-template-columns: 1fr;
    gap: var(--el-space-lg);
  }

  .create-app-dialog__promotion {
    min-height: 144px;
    padding: var(--el-space-2xl);
  }

  .create-app-dialog__promotion-content {
    strong {
      font-size: var(--el-font-size-medium);
    }

    span {
      font-size: var(--el-font-size-base);
    }
  }
}
</style>
