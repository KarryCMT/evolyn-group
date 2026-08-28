<script setup lang="ts">
import type { MessagePreference, ReminderRecipient } from './messageCenter.types';
import { RiCloseFill } from '@remixicon/vue';
import { computed, ref, shallowRef, watch } from 'vue';

defineOptions({ name: 'MessageRecipientSelectorDialog' });

const props = defineProps<{
  /** 当前编辑的事件码（空串表示关闭） */
  eventCode: string;
  preferences: MessagePreference[];
  recipients: ReminderRecipient[];
}>();

const emit = defineEmits<{
  /** 全量替换该事件接收规则（保存动作） */
  submit: [eventCode: string, next: { kind: string; recipientId?: number }[]];
}>();

/** 事件码非空即打开：el-dialog 需要 boolean v-model，经 computed 转换 */
const eventCodeModel = defineModel<string>({ default: '' });
const dialogVisible = computed<boolean>({
  get: () => eventCodeModel.value !== '',
  set: (value) => {
    if (!value) eventCodeModel.value = '';
  },
});

const dynamicKinds = [
  { kind: 'event_actor', label: '创建者' },
  { kind: 'event_audience', label: '指定成员' },
  { kind: 'tenant_admin', label: '系统管理员' },
] as const;

const selectedKinds = ref<string[]>([]);
const selectedCustomIds = ref<number[]>([]);
const saving = shallowRef(false);

const activePreference = computed(() =>
  props.preferences.find((item) => item.code === props.eventCode),
);
/** 自定义联系人仅用于邮件/短信渠道（服务端同样限制站内信不含外部联系人） */
const customRecipients = computed(() => props.recipients);

/** 打开时以该事件当前有效接收规则初始化勾选。 */
watch(
  eventCodeModel,
  (code) => {
    if (!code) return;
    const preference = props.preferences.find((item) => item.code === code);
    selectedKinds.value =
      preference?.recipients
        .filter((item) => item.kind !== 'custom_recipient')
        .map((item) => item.kind) ?? [];
    selectedCustomIds.value =
      preference?.recipients
        .filter((item) => item.kind === 'custom_recipient')
        .map((item) => item.recipientId ?? 0)
        .filter(Boolean) ?? [];
  },
  { immediate: true },
);

/** 提交全量替换：动态规则 + 自定义联系人一次上送（服务端校验归属）。 */
async function submit() {
  if (saving.value || !props.eventCode) return;
  const next: { kind: string; recipientId?: number }[] = selectedKinds.value.map((kind) => ({
    kind,
  }));
  for (const id of selectedCustomIds.value)
    next.push({ kind: 'custom_recipient', recipientId: id });
  saving.value = true;
  try {
    emit('submit', props.eventCode, next);
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <el-dialog
    v-model="dialogVisible"
    class="message-recipient-selector"
    title="接收对象"
    width="560px"
    :close-on-click-modal="false"
    :show-close="false"
  >
    <template #header="{ close, titleId, titleClass }">
      <div class="message-recipient-selector__header">
        <h2 :id="titleId" :class="titleClass">
          接收对象{{ activePreference ? `：${activePreference.label}` : '' }}
        </h2>
        <el-button
          class="message-recipient-selector__close"
          :icon="RiCloseFill"
          text
          circle
          aria-label="关闭接收对象弹窗"
          @click="close"
        />
      </div>
    </template>

    <el-scrollbar class="message-recipient-selector__body" max-height="420px">
      <el-checkbox-group v-model="selectedKinds">
        <el-checkbox v-for="item in dynamicKinds" :key="item.kind" :value="item.kind">
          {{ item.label }}
        </el-checkbox>
      </el-checkbox-group>

      <p class="message-recipient-selector__section">自定义提醒对象（仅邮件/短信渠道）</p>
      <template v-if="customRecipients.length">
        <el-checkbox-group v-model="selectedCustomIds">
          <el-checkbox v-for="item in customRecipients" :key="item.id" :value="item.id">
            {{ item.name }}（{{ item.mobile || item.email }}）
          </el-checkbox>
        </el-checkbox-group>
      </template>
      <p v-else class="message-recipient-selector__empty">
        暂无自定义提醒对象，可在「提醒对象管理」中添加
      </p>
    </el-scrollbar>

    <template #footer>
      <el-button @click="dialogVisible = false"> 取消 </el-button>
      <el-button :loading="saving" type="primary" @click="submit"> 保存 </el-button>
    </template>
  </el-dialog>
</template>

<!-- Dialog 传送至 body，以独立类限定全局覆盖（不影响其他弹窗）。 -->
<style lang="scss">
.message-recipient-selector.el-dialog {
  overflow: hidden;
  border-radius: var(--el-border-radius-large);
}

.message-recipient-selector .el-dialog__header {
  height: 56px;
  margin: 0;
  padding: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.message-recipient-selector__header {
  display: flex;
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--el-space-xl) 0 var(--el-space-3xl);
}

.message-recipient-selector__header .el-dialog__title {
  margin: 0;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 18px;
  font-weight: 700;
  line-height: 26px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.message-recipient-selector__close.el-button {
  min-width: 32px;
  width: 32px;
  height: 32px;
  padding: 0;
  border-radius: var(--el-border-radius-medium);
  color: var(--el-text-color-regular);
  font-size: var(--el-font-size-medium);
}

.message-recipient-selector__close.el-button:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.message-recipient-selector__body {
  padding: var(--el-space-2xl) var(--el-space-3xl) var(--el-space-md);

  .el-checkbox {
    display: flex;
    height: 36px;
    margin-right: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
  }
}

.message-recipient-selector__section {
  margin: var(--el-space-lg) 0 var(--el-space-md);
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-base);
  line-height: 22px;
}

.message-recipient-selector__empty {
  margin: 0;
  color: var(--el-text-color-placeholder);
  font-size: var(--el-font-size-base);
  line-height: 22px;
}

.message-recipient-selector .el-dialog__footer {
  padding: var(--el-space-lg) var(--el-space-3xl) var(--el-space-xl);
  border-top: 1px solid var(--el-border-color-lighter);
}

.message-recipient-selector .el-button {
  min-width: 76px;
  height: 36px;
}

@media (width <= 767px) {
  .message-recipient-selector .el-dialog__header {
    height: 52px;
  }
}
</style>
