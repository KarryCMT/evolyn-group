<script setup lang="ts">
import { RiCloseFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, reactive, shallowRef } from 'vue';

defineOptions({ name: 'DataPushEditor' });

const props = defineProps<{
  formName: string;
}>();

const emit = defineEmits<{
  close: [];
  save: [name: string];
}>();

type PushType = 'form' | 'workflow';

interface PushDraft {
  name: string;
  targetServer: string;
  endpoint: string;
  secret: string;
  type: PushType;
  remark: string;
}

interface PushEventOption {
  value: string;
  label: string;
  example?: boolean;
}

const draft = reactive<PushDraft>({
  name: '未命名数据推送',
  targetServer: 'custom',
  endpoint: '',
  secret: '',
  type: 'form',
  remark: '',
});
const endpointInvalid = shallowRef(false);
const formEvents = shallowRef(['create', 'update', 'delete', 'restore']);
const workflowEvents = shallowRef<string[]>([]);

const eventOptions = computed<PushEventOption[]>(() =>
  draft.type === 'form'
    ? [
        { value: 'create', label: '有新数据提交时' },
        { value: 'update', label: '有数据被修改时' },
        { value: 'delete', label: '有数据被删除时' },
        { value: 'restore', label: '有数据被恢复时' },
        { value: 'structure', label: '表单结构有变化时' },
      ]
    : [
        { value: 'status', label: '流程状态变更时', example: true },
        { value: 'todo', label: '流程待办变更时', example: true },
        { value: 'cc', label: '产生抄送时', example: true },
      ],
);

const selectedEvents = computed({
  get: () => (draft.type === 'form' ? formEvents.value : workflowEvents.value),
  set: (value: string[]) => {
    if (draft.type === 'form') formEvents.value = value;
    else workflowEvents.value = value;
  },
});

function selectType(type: PushType) {
  draft.type = type;
}

function generateSecret() {
  const nonce = Math.random().toString(36).slice(2, 12);
  draft.secret = `dp_${nonce}${Date.now().toString(36)}`;
  ElMessage.success('Secret 已生成');
}

function testServer() {
  endpointInvalid.value = !draft.endpoint.trim();
  if (endpointInvalid.value) {
    ElMessage.warning('请先填写服务器地址');
    return;
  }
  ElMessage.success('服务器连接测试通过');
}

function save() {
  endpointInvalid.value = !draft.endpoint.trim();
  if (endpointInvalid.value) {
    ElMessage.warning('请填写服务器地址后再保存');
    return;
  }
  if (!selectedEvents.value.length) {
    ElMessage.warning('请至少选择一个推送事件');
    return;
  }
  emit('save', draft.name.trim() || '未命名数据推送');
}

function showExample() {
  ElMessage.info('事件推送示例将在帮助文档中提供');
}
</script>

<template>
  <section class="data-push-editor" aria-label="数据推送配置">
    <header class="data-push-editor__header">
      <h1 class="data-push-editor__title">数据推送</h1>
      <button
        class="data-push-editor__close"
        type="button"
        aria-label="关闭"
        @click="emit('close')"
      >
        <RiCloseFill aria-hidden="true" />
      </button>
    </header>

    <el-scrollbar class="data-push-editor__scrollbar">
      <div class="data-push-editor__canvas">
        <section
          class="data-push-editor__form-card"
          :aria-label="`${props.formName}的数据推送配置`"
        >
          <div class="data-push-editor__field data-push-editor__field--narrow">
            <label class="data-push-editor__label" for="data-push-name">数据推送名称</label>
            <el-input id="data-push-name" v-model="draft.name" />
          </div>

          <div class="data-push-editor__field data-push-editor__field--narrow">
            <label class="data-push-editor__label" for="data-push-target">目标服务器</label>
            <el-select
              id="data-push-target"
              v-model="draft.targetServer"
              class="data-push-editor__control"
            >
              <el-option label="自定义服务器" value="custom" />
            </el-select>
          </div>

          <div class="data-push-editor__field">
            <label class="data-push-editor__label" for="data-push-endpoint">服务器地址</label>
            <div class="data-push-editor__inline-control">
              <el-input
                id="data-push-endpoint"
                v-model="draft.endpoint"
                :class="{ 'data-push-editor__input--invalid': endpointInvalid }"
                placeholder="https://example.com/webhook"
                @input="endpointInvalid = false"
              />
              <el-button type="primary" @click="testServer"> 服务器连接测试 </el-button>
            </div>
          </div>

          <div class="data-push-editor__field">
            <label class="data-push-editor__label" for="data-push-secret">Secret</label>
            <div class="data-push-editor__inline-control">
              <el-input id="data-push-secret" v-model="draft.secret" />
              <el-button type="primary" @click="generateSecret"> 生成 Secret </el-button>
            </div>
          </div>

          <div class="data-push-editor__field">
            <p class="data-push-editor__label">推送类型</p>
            <div class="data-push-editor__types" role="radiogroup" aria-label="推送类型">
              <button
                class="data-push-editor__type-card"
                :class="{ 'data-push-editor__type-card--selected': draft.type === 'form' }"
                type="button"
                role="radio"
                :aria-checked="draft.type === 'form'"
                @click="selectType('form')"
              >
                <strong>表单变更时推送</strong>
                <span>包括数据变更、表单结构变更</span>
              </button>
              <button
                class="data-push-editor__type-card"
                :class="{ 'data-push-editor__type-card--selected': draft.type === 'workflow' }"
                type="button"
                role="radio"
                :aria-checked="draft.type === 'workflow'"
                @click="selectType('workflow')"
              >
                <strong>流程变更时推送</strong>
                <span>包括流程状态变更、待办变更等</span>
              </button>
            </div>
          </div>

          <div class="data-push-editor__field data-push-editor__events">
            <p class="data-push-editor__label">推送事件</p>
            <template v-if="draft.type === 'form'">
              <p class="data-push-editor__event-intro">
                <strong>数据事件</strong>
                <span>表单数据发生变更时，推送变更后的数据</span>
                <button type="button" @click="showExample">查看样例</button>
              </p>
              <el-checkbox-group v-model="selectedEvents" class="data-push-editor__event-list">
                <el-checkbox
                  v-for="event in eventOptions.slice(0, 4)"
                  :key="event.value"
                  :value="event.value"
                >
                  {{ event.label }}
                </el-checkbox>
              </el-checkbox-group>
              <p class="data-push-editor__event-intro">
                <strong>表单事件</strong>
                <span>表单保存或字段别名变更时，推送最新表单结构</span>
                <button type="button" @click="showExample">查看样例</button>
              </p>
              <el-checkbox-group v-model="selectedEvents" class="data-push-editor__event-list">
                <el-checkbox :value="eventOptions[4].value">
                  {{ eventOptions[4].label }}
                </el-checkbox>
              </el-checkbox-group>
            </template>
            <el-checkbox-group v-else v-model="selectedEvents" class="data-push-editor__event-list">
              <el-checkbox v-for="event in eventOptions" :key="event.value" :value="event.value">
                {{ event.label }}
                <button
                  v-if="event.example"
                  class="data-push-editor__example-link"
                  type="button"
                  @click.stop="showExample"
                >
                  查看样例
                </button>
              </el-checkbox>
            </el-checkbox-group>
          </div>

          <div class="data-push-editor__field">
            <label class="data-push-editor__label" for="data-push-remark">备注</label>
            <el-input
              id="data-push-remark"
              v-model="draft.remark"
              type="textarea"
              :rows="4"
              placeholder="请输入备注"
            />
          </div>
        </section>
      </div>
    </el-scrollbar>

    <footer class="data-push-editor__footer">
      <el-button type="primary" class="data-push-editor__save" @click="save"> 保存 </el-button>
    </footer>
  </section>
</template>

<style scoped lang="scss">
.data-push-editor {
  position: fixed;
  z-index: 1000;
  top: 52px;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  min-width: 720px;
  overflow: hidden;
  flex-direction: column;
  background: var(--el-fill-color-light);

  &__header {
    position: relative;
    display: flex;
    min-height: 80px;
    padding: 0 28px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    align-items: center;
    justify-content: center;
    background: var(--el-bg-color);
  }

  &__title {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 22px;
    font-weight: 650;
    line-height: 30px;
  }

  &__close {
    position: absolute;
    top: 24px;
    right: 28px;
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    border: 0;
    border-radius: var(--el-border-radius-base);
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;

    svg {
      width: 22px;
      height: 22px;
    }

    &:hover {
      color: var(--el-text-color-primary);
      background: var(--el-fill-color);
    }
  }

  &__scrollbar {
    min-height: 0;
    flex: 1;
  }

  &__canvas {
    box-sizing: border-box;
    min-height: 100%;
    padding: 28px 24px 44px;
  }

  &__form-card {
    box-sizing: border-box;
    width: min(100%, 960px);
    min-height: 860px;
    margin: 0 auto;
    padding: 28px 42px 48px;
    border-radius: 10px;
    background: var(--el-bg-color);
  }

  &__field {
    margin-top: 26px;

    &:first-child {
      margin-top: 0;
    }

    &--narrow {
      width: 330px;
    }
  }

  &__label {
    display: block;
    margin: 0 0 10px;
    color: var(--el-text-color-regular);
    font-size: 16px;
    line-height: 24px;
  }

  &__control {
    width: 330px;
  }

  &__inline-control {
    display: flex;
    width: 100%;
    align-items: center;
    gap: 26px;

    :deep(.el-input) {
      max-width: 635px;
    }

    :deep(.el-button) {
      min-width: 206px;
    }
  }

  &__input--invalid {
    :deep(.el-input__wrapper) {
      box-shadow: 0 0 0 1px var(--el-color-danger) inset;
    }
  }

  &__types {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }

  &__type-card {
    display: flex;
    min-height: 110px;
    padding: 20px;
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-primary);
    cursor: pointer;
    background: var(--el-bg-color);
    flex-direction: column;
    font: inherit;
    gap: 8px;

    strong {
      font-size: 17px;
      line-height: 24px;
    }

    span {
      color: var(--el-text-color-secondary);
      font-size: 15px;
      line-height: 22px;
    }

    &:hover {
      border-color: var(--el-color-primary-light-3);
      background: var(--el-color-primary-light-9);
    }

    &--selected {
      border-color: var(--el-color-primary);
      box-shadow: 0 0 0 1px var(--el-color-primary) inset;
    }
  }

  &__events {
    margin-top: 28px;
  }

  &__event-intro {
    display: flex;
    margin: 0 0 8px;
    align-items: center;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
    gap: 10px;

    strong {
      color: var(--el-text-color-primary);
      font-size: 16px;
    }

    button,
    .data-push-editor__example-link {
      padding: 0;
      border: 0;
      color: var(--el-color-primary);
      cursor: pointer;
      background: transparent;
      font: inherit;

      &:hover {
        color: var(--el-color-primary-light-3);
      }
    }
  }

  &__event-list {
    display: flex;
    margin: 0 0 16px;
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;

    :deep(.el-checkbox) {
      height: 24px;
      margin-right: 0;
      color: var(--el-text-color-regular);
      font-size: 16px;
    }
  }

  &__example-link {
    margin-left: 8px;
  }

  &__footer {
    display: flex;
    min-height: 76px;
    border-top: 1px solid var(--el-border-color-lighter);
    align-items: center;
    justify-content: center;
    background: var(--el-bg-color);
  }

  &__save {
    min-width: 166px;
    height: 42px;
    font-size: 16px;
  }
}

@media (max-width: 1000px) {
  .data-push-editor {
    &__form-card {
      padding-right: 30px;
      padding-left: 30px;
    }

    &__inline-control {
      :deep(.el-button) {
        min-width: 150px;
      }
    }
  }
}
</style>
