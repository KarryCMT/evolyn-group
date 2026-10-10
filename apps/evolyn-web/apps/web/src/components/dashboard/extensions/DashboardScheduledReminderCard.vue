<script setup lang="ts">
import type { MemberListItemDto } from '~/api/member';
import { RiAddLine, RiCloseLine, RiQuestionFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, reactive, shallowRef } from 'vue';
import MemberPickerDialog from '~/components/form/MemberPickerDialog.vue';

defineOptions({ name: 'DashboardScheduledReminderCard' });

const enabled = shallowRef(false);
const pickerVisible = shallowRef(false);
const recipients = shallowRef<MemberListItemDto[]>([]);
const settings = reactive({
  startsAt: '',
  repeat: 'once',
  message: '已到提醒时间，请及时处理',
  email: true,
  wechat: false,
});

const selectedIds = computed(() => recipients.value.map((member) => member.memberCode));

const repeatOptions = [
  { value: 'once', label: '只提醒一次' },
  { value: 'daily', label: '每天' },
  { value: 'weekly', label: '每周' },
  { value: 'monthly', label: '每月' },
];

function saveSettings(): void {
  if (!settings.startsAt) {
    ElMessage.warning('请设置开始提醒时间');
    return;
  }
  if (!recipients.value.length) {
    ElMessage.warning('请选择至少一名被提醒人');
    return;
  }
  ElMessage.success('定时提醒设置已保存');
}

function removeRecipient(memberCode: string): void {
  recipients.value = recipients.value.filter((member) => member.memberCode !== memberCode);
}
</script>

<template>
  <section class="dashboard-reminder-card" aria-labelledby="scheduled-reminder-title">
    <header class="dashboard-reminder-card__header">
      <h2 id="scheduled-reminder-title">
        定时提醒
      </h2>
      <p>定时将仪表盘推送给成员查看，提高工作效率</p>
    </header>

    <div class="dashboard-reminder-card__body">
      <el-switch v-model="enabled" aria-label="启用定时提醒" />

      <div v-if="enabled" class="dashboard-reminder-card__settings">
        <div class="dashboard-reminder-card__field dashboard-reminder-card__field--compact">
          <label for="reminder-starts-at">开始提醒时间</label>
          <el-date-picker
            id="reminder-starts-at"
            v-model="settings.startsAt"
            class="dashboard-reminder-card__control"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            format="YYYY-MM-DD HH:mm"
            placeholder="请设置开始提醒时间"
          />
        </div>

        <div class="dashboard-reminder-card__field dashboard-reminder-card__field--compact">
          <label for="reminder-repeat">重复类型</label>
          <el-select
            id="reminder-repeat"
            v-model="settings.repeat"
            class="dashboard-reminder-card__control"
            aria-label="重复类型"
          >
            <el-option
              v-for="option in repeatOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
        </div>

        <div class="dashboard-reminder-card__field">
          <label class="dashboard-reminder-card__label-with-help">
            被提醒人
            <el-tooltip content="当前支持从通讯录选择成员" placement="top">
              <RiQuestionFill aria-label="被提醒人说明" />
            </el-tooltip>
          </label>
          <button
            class="dashboard-reminder-card__recipient-picker"
            type="button"
            @click="pickerVisible = true"
          >
            <template v-if="recipients.length">
              <span
                v-for="member in recipients"
                :key="member.memberCode"
                class="dashboard-reminder-card__recipient"
              >
                <i>{{ member.name.slice(0, 1) }}</i>
                {{ member.name }}
                <RiCloseLine @click.stop="removeRecipient(member.memberCode)" />
              </span>
            </template>
            <span v-else class="dashboard-reminder-card__recipient-placeholder">
              <RiAddLine />选择成员或部门
            </span>
          </button>
        </div>

        <div class="dashboard-reminder-card__field">
          <label for="reminder-message">提醒文字</label>
          <el-input id="reminder-message" v-model="settings.message" maxlength="200" />
        </div>

        <fieldset class="dashboard-reminder-card__channels">
          <legend>提醒方式</legend>
          <div class="dashboard-reminder-card__channel-group">
            <span class="dashboard-reminder-card__channel-label">
              内部成员
              <el-tooltip content="已选择成员将按以下方式收到提醒" placement="top">
                <RiQuestionFill aria-label="内部成员提醒说明" />
              </el-tooltip>
            </span>
            <div class="dashboard-reminder-card__checkboxes">
              <el-checkbox v-model="settings.email">
                邮箱消息
              </el-checkbox>
              <el-checkbox v-model="settings.wechat">
                微信提醒
              </el-checkbox>
            </div>
          </div>
          <div class="dashboard-reminder-card__channel-group is-disabled">
            <span class="dashboard-reminder-card__channel-label">互联对接人</span>
            <div class="dashboard-reminder-card__checkboxes">
              <el-checkbox :model-value="true" disabled>
                邮箱消息
              </el-checkbox>
              <el-checkbox :model-value="true" disabled>
                集成模式
              </el-checkbox>
            </div>
          </div>
        </fieldset>

        <footer class="dashboard-reminder-card__footer">
          <el-button type="primary" class="dashboard-reminder-card__save" @click="saveSettings">
            保存
          </el-button>
        </footer>
      </div>
    </div>

    <MemberPickerDialog
      v-model="pickerVisible"
      multiple
      :selected-ids="selectedIds"
      @confirm="recipients = $event"
    />
  </section>
</template>

<style scoped>
/* Vue 的 :deep() 用于统一 Element Plus 控件宽度与复选框间距。 */
/* stylelint-disable selector-pseudo-class-no-unknown */
.dashboard-reminder-card {
  overflow: hidden;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 2px;
  box-shadow: 0 2px 8px rgb(31 45 61 / 6%);
}

.dashboard-reminder-card__header {
  display: flex;
  min-height: 64px;
  padding: 0 28px;
  align-items: center;
  gap: 14px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.dashboard-reminder-card__header h2,
.dashboard-reminder-card__header p {
  margin: 0;
}

.dashboard-reminder-card__header h2 {
  flex: 0 0 auto;
  font-size: 16px;
  font-weight: 650;
  color: var(--el-text-color-primary);
}

.dashboard-reminder-card__header p {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.dashboard-reminder-card__body {
  min-height: 84px;
  padding: 26px 28px;
}

.dashboard-reminder-card__settings {
  margin-top: 14px;
}

.dashboard-reminder-card__field {
  width: 100%;
  margin-top: 18px;
}

.dashboard-reminder-card__field--compact {
  width: min(380px, 100%);
}

.dashboard-reminder-card__field > label,
.dashboard-reminder-card__channels legend {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}

.dashboard-reminder-card__label-with-help,
.dashboard-reminder-card__channel-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.dashboard-reminder-card__label-with-help svg,
.dashboard-reminder-card__channel-label svg {
  width: 15px;
  height: 15px;
  color: var(--el-text-color-placeholder);
}

.dashboard-reminder-card__control {
  width: 100%;
}

.dashboard-reminder-card__recipient-picker {
  display: flex;
  width: min(760px, 100%);
  min-height: 116px;
  padding: 14px;
  align-items: flex-start;
  align-content: flex-start;
  justify-content: flex-start;
  flex-wrap: wrap;
  gap: 8px;
  font: inherit;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: var(--el-bg-color);
  border: 1px dashed var(--el-border-color);
  border-radius: var(--el-border-radius-base);
}

.dashboard-reminder-card__recipient-picker:hover {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.dashboard-reminder-card__recipient-placeholder {
  display: inline-flex;
  width: 100%;
  height: 84px;
  align-items: center;
  justify-content: center;
  gap: 5px;
  color: var(--el-text-color-regular);
}

.dashboard-reminder-card__recipient-placeholder svg {
  width: 18px;
  height: 18px;
}

.dashboard-reminder-card__recipient {
  display: inline-flex;
  height: 30px;
  padding: 0 9px 0 4px;
  align-items: center;
  gap: 6px;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  border-radius: var(--el-border-radius-base);
}

.dashboard-reminder-card__recipient i {
  display: inline-grid;
  width: 22px;
  height: 22px;
  color: var(--el-color-white);
  background: var(--el-color-primary);
  border-radius: 50%;
  place-items: center;
  font-size: 11px;
  font-style: normal;
}

.dashboard-reminder-card__recipient svg {
  width: 15px;
  height: 15px;
}

.dashboard-reminder-card__channels {
  padding: 0;
  margin: 20px 0 0;
  border: 0;
}

.dashboard-reminder-card__channel-group + .dashboard-reminder-card__channel-group {
  margin-top: 14px;
}

.dashboard-reminder-card__channel-group.is-disabled {
  color: var(--el-text-color-disabled);
}

.dashboard-reminder-card__channel-label {
  margin-bottom: 6px;
  font-size: 14px;
}

.dashboard-reminder-card__checkboxes {
  display: flex;
  min-height: 28px;
  align-items: center;
  gap: 28px;
}

.dashboard-reminder-card__checkboxes :deep(.el-checkbox) {
  margin-right: 0;
}

.dashboard-reminder-card__footer {
  margin-top: 26px;
  padding-top: 22px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.dashboard-reminder-card__save {
  min-width: 174px;
}

@media (width <= 640px) {
  .dashboard-reminder-card__header {
    padding: 14px 18px;
    align-items: flex-start;
    flex-direction: column;
    gap: 4px;
  }

  .dashboard-reminder-card__body {
    padding: 22px 18px;
  }

  .dashboard-reminder-card__checkboxes {
    align-items: flex-start;
    flex-direction: column;
    gap: 0;
  }
}
</style>
