<script setup lang="ts">
import {
  RiArrowRightSLine,
  RiCloseFill,
  RiLightbulbFlashFill,
  RiRefreshFill,
  RiThumbUpFill,
} from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import customApplicationBackground from '~/assets/images/customer_bg.png';
import templateCenterBackground from '~/assets/images/template-center-banner_bg.png';
import ApplicationStarterCard from './ApplicationStarterCard.vue';
import ApplicationTemplateCard from './ApplicationTemplateCard.vue';
import BlankApplicationDialog, { type BlankApplicationDraft } from './BlankApplicationDialog.vue';
import {
  applicationStarters,
  applicationTemplateBatches,
  type ApplicationStarter,
  type ApplicationTemplate,
} from './applicationTemplateCatalog';

defineOptions({ name: 'CreateApplicationDialog' });

defineProps<{
  /** 空白应用异步创建处理：resolve true 关闭两级弹窗，false 保持开启（失败保留填写内容） */
  submitBlank: (draft: BlankApplicationDraft) => Promise<boolean>;
}>();

const emit = defineEmits<{
  selectStarter: [starter: Exclude<ApplicationStarter, { id: 'blank' }>];
  selectTemplate: [template: ApplicationTemplate];
  openTemplateCenter: [];
  requestCustomization: [];
}>();

const visible = defineModel<boolean>({ default: false });
const activeTemplateBatch = shallowRef(0);
const blankApplicationVisible = shallowRef(false);
const visibleTemplates = computed(
  () => applicationTemplateBatches[activeTemplateBatch.value] ?? [],
);

function selectStarter(starter: ApplicationStarter) {
  if (starter.id === 'blank') {
    blankApplicationVisible.value = true;
    return;
  }

  emit('selectStarter', starter);
}

// 关闭一级弹窗时一并收起二级表单，避免传送到 body 的子弹窗失去上下文。
// 提交进行中由二级表单自身禁用关闭入口，此处不强制打断请求。
watch(visible, (isVisible) => {
  if (!isVisible) blankApplicationVisible.value = false;
});

// 二级表单异步提交成功后才关闭两级弹窗；失败由处理方提示并保留表单
function handleBlankSuccess() {
  blankApplicationVisible.value = false;
  visible.value = false;
}

function showNextTemplates() {
  activeTemplateBatch.value = (activeTemplateBatch.value + 1) % applicationTemplateBatches.length;
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="create-application-dialog"
    width="880px"
    top="8vh"
    :show-close="false"
    :close-on-click-modal="false"
    append-to-body
  >
    <template #header>
      <header class="create-application-dialog__header">
        <h2 class="create-application-dialog__heading">新建应用</h2>
        <button
          class="create-application-dialog__close"
          type="button"
          aria-label="关闭新建应用"
          @click="visible = false"
        >
          <RiCloseFill />
        </button>
      </header>
    </template>

    <div class="create-application-dialog__body">
      <section
        class="create-application-dialog__section"
        aria-labelledby="application-starter-heading"
      >
        <div class="create-application-dialog__section-heading">
          <RiLightbulbFlashFill aria-hidden="true" />
          <h3 id="application-starter-heading">猜你想要</h3>
        </div>
        <div class="create-application-dialog__starter-grid">
          <ApplicationStarterCard
            v-for="starter in applicationStarters"
            :key="starter.id"
            :starter="starter"
            @select="selectStarter"
          />
        </div>
      </section>

      <section
        class="create-application-dialog__section"
        aria-labelledby="application-template-heading"
      >
        <div
          class="create-application-dialog__section-heading create-application-dialog__section-heading--split"
        >
          <span class="create-application-dialog__section-title">
            <RiThumbUpFill aria-hidden="true" />
            <h3 id="application-template-heading">热门模板</h3>
          </span>
          <button
            class="create-application-dialog__refresh"
            type="button"
            @click="showNextTemplates"
          >
            <RiRefreshFill aria-hidden="true" />
            换一批
          </button>
        </div>
        <div class="create-application-dialog__template-grid">
          <ApplicationTemplateCard
            v-for="template in visibleTemplates"
            :key="template.id"
            :template="template"
            @select="emit('selectTemplate', $event)"
          />
        </div>
      </section>

      <section
        class="create-application-dialog__section"
        aria-labelledby="application-more-heading"
      >
        <div class="create-application-dialog__section-heading">
          <RiLightbulbFlashFill aria-hidden="true" />
          <h3 id="application-more-heading">没有找到想要的模板？试试下面</h3>
        </div>
        <div class="create-application-dialog__more-grid">
          <button
            class="create-application-dialog__promotion create-application-dialog__promotion--templates"
            type="button"
            :style="{ backgroundImage: `url(${templateCenterBackground})` }"
            @click="emit('openTemplateCenter')"
          >
            <span class="create-application-dialog__promotion-content">
              <strong>模板中心</strong>
              <span>海量优质模板，总有一款适合你</span>
            </span>
            <RiArrowRightSLine aria-hidden="true" />
          </button>
          <button
            class="create-application-dialog__promotion create-application-dialog__promotion--custom"
            type="button"
            :style="{ backgroundImage: `url(${customApplicationBackground})` }"
            @click="emit('requestCustomization')"
          >
            <span class="create-application-dialog__promotion-content">
              <strong>定制应用</strong>
              <span>官方团队为你个性化定制</span>
            </span>
            <RiArrowRightSLine aria-hidden="true" />
          </button>
        </div>
      </section>
    </div>
  </el-dialog>
  <BlankApplicationDialog
    v-model="blankApplicationVisible"
    :submit="submitBlank"
    @success="handleBlankSuccess"
  />
</template>

<!-- 弹窗会传送至 body，样式须通过唯一块类限定。 -->
<style lang="scss">
.create-application-dialog.el-dialog {
  display: flex;
  max-width: calc(100vw - 32px);
  height: 665px;
  max-height: calc(100vh - 32px);
  margin-bottom: 0;
  overflow: hidden;
  flex-direction: column;
  border-radius: 18px;
}

.create-application-dialog .el-dialog__header {
  flex: 0 0 auto;
  padding: 0;
  margin: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.create-application-dialog .el-dialog__body {
  flex: 1;
  min-height: 0;
  padding: 0;
  overflow: hidden;
}

.create-application-dialog__header {
  display: flex;
  height: 56px;
  padding: 0 28px 0 40px;
  align-items: center;
  justify-content: space-between;
}

.create-application-dialog__heading {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.15;
}

.create-application-dialog__close,
.create-application-dialog__refresh {
  display: inline-flex;
  padding: 0;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  background: transparent;
  border: 0;
}

.create-application-dialog__close {
  width: 32px;
  height: 32px;
  color: var(--el-text-color-regular);
  border-radius: var(--el-border-radius-base);
  font-size: 22px;
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

.create-application-dialog__body {
  box-sizing: border-box;
  display: flex;
  height: 100%;
  min-height: 0;
  padding: 18px 24px;
  flex-direction: column;
  gap: 18px;
}

.create-application-dialog__section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.create-application-dialog__section-heading,
.create-application-dialog__section-title {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: var(--el-text-color-primary);

  > svg {
    color: var(--el-text-color-regular);
    font-size: 24px;
  }

  h3 {
    margin: 0;
    font-size: 17px;
    font-weight: 650;
    letter-spacing: -0.01em;
    line-height: 1.3;
  }

  &--split {
    justify-content: space-between;
  }
}

.create-application-dialog__refresh {
  height: 32px;
  padding: 0 5px;
  gap: 5px;
  color: var(--el-color-primary);
  border-radius: var(--el-border-radius-base);
  font-size: 14px;
  transition: background-color 0.2s ease;

  svg {
    font-size: 21px;
  }

  &:hover {
    background: var(--el-color-primary-light-9);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }
}

.create-application-dialog__starter-grid,
.create-application-dialog__template-grid,
.create-application-dialog__more-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.create-application-dialog__promotion {
  display: flex;
  min-height: 108px;
  padding: 18px 22px;
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
  border-radius: 14px;
  box-shadow: 0 2px 6px rgb(31 35 41 / 8%);
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
    margin-top: 2px;
    font-size: 28px;
  }
}

.create-application-dialog__promotion-content {
  display: flex;
  max-width: 340px;
  flex-direction: column;
  gap: 6px;

  strong {
    font-size: 19px;
    font-weight: 700;
    line-height: 1.2;
  }

  span {
    font-size: 13px;
    line-height: 1.5;
  }
}

@media (max-width: 1100px) {
  .create-application-dialog__starter-grid,
  .create-application-dialog__template-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .create-application-dialog__more-grid {
    grid-template-columns: 1fr;
  }

  .create-application-dialog__promotion--templates,
  .create-application-dialog__promotion--custom {
    grid-column: auto;
  }
}

@media (max-width: 720px) {
  .create-application-dialog.el-dialog {
    width: 100vw !important;
    height: 100vh;
    border-radius: 0;
  }

  .create-application-dialog__header {
    height: 52px;
    padding: 0 12px 0 20px;
  }

  .create-application-dialog__heading {
    font-size: 18px;
    line-height: 26px;
  }

  .create-application-dialog__body {
    padding: 24px 18px 32px;
    gap: 30px;
  }

  .create-application-dialog__section {
    gap: 16px;
  }

  .create-application-dialog__section-heading,
  .create-application-dialog__section-title {
    gap: 8px;

    > svg {
      font-size: 20px;
    }

    h3 {
      font-size: 18px;
    }
  }

  .create-application-dialog__starter-grid,
  .create-application-dialog__template-grid {
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .create-application-dialog__promotion {
    min-height: 144px;
    padding: 22px;
  }

  .create-application-dialog__promotion-content {
    strong {
      font-size: 20px;
    }

    span {
      font-size: 14px;
    }
  }
}
</style>
