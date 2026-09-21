<script setup lang="ts">
import { RiCheckLine } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, reactive, shallowRef } from 'vue';
import EnterpriseSettingRow from './EnterpriseSettingRow.vue';
import EnterpriseSettingsSection from './EnterpriseSettingsSection.vue';
import LoginStyleDialog from './LoginStyleDialog.vue';
import ReminderSuppressionDialog from './ReminderSuppressionDialog.vue';
import SsoSettingsDialog from './SsoSettingsDialog.vue';

defineOptions({ name: 'EnterpriseSettingsPanel' });

// 当前阶段先在页面内维护交互状态，后端接口接入后可直接替换为配置查询与保存结果。
const settings = reactive({
  watermark: false,
  externalAccountLogin: false,
  aiConnection: true,
  customLogin: false,
  enterpriseStyle: false,
  language: 'zh-TW',
  timezone: '(GMT+8:00) Asia/Taipei',
  aiCapability: true,
  wechatIntegration: false,
});

const ssoDialogVisible = shallowRef(false);
const loginStyleDialogVisible = shallowRef(false);
const reminderDialogVisible = shallowRef(false);
const timezoneDialogVisible = shallowRef(false);
const themePopoverVisible = shallowRef(false);
const selectedTheme = shallowRef('#0bb6aa');
const pendingTimezone = shallowRef(settings.timezone);

const themeColors = [
  '#0bb6aa',
  '#f45358',
  '#ff7d1a',
  '#f3a81d',
  '#35af50',
  '#16abd0',
  '#2672ef',
  '#516bd9',
  '#7650e6',
  '#b747d5',
  '#e3457a',
  '#5c6b80',
];

const activeThemeStyle = computed(() => ({ '--enterprise-accent': selectedTheme.value }));

function learnMore(topic: string) {
  ElMessage.info(`${topic}帮助文档正在建设中`);
}

function showPlannedSetting(label: string) {
  ElMessage.info(`${label}配置面板将在后续版本接入`);
}

function toggleSso(enabled: boolean | string | number) {
  settings.externalAccountLogin = Boolean(enabled);
  if (settings.externalAccountLogin) ssoDialogVisible.value = true;
}

function toggleCustomLogin(enabled: boolean | string | number) {
  settings.customLogin = Boolean(enabled);
  if (settings.customLogin) loginStyleDialogVisible.value = true;
}

function toggleEnterpriseStyle(enabled: boolean | string | number) {
  settings.enterpriseStyle = Boolean(enabled);
  if (settings.enterpriseStyle) themePopoverVisible.value = true;
}

function saveTheme() {
  themePopoverVisible.value = false;
  ElMessage.success('企业主题色已保存');
}

function saveTimezone() {
  settings.timezone = pendingTimezone.value;
  timezoneDialogVisible.value = false;
  ElMessage.success('系统时区已修改');
}
</script>

<template>
  <div class="enterprise-settings-panel" :style="activeThemeStyle">
    <el-scrollbar class="enterprise-settings-panel__scrollbar">
      <div class="enterprise-settings-panel__content">
        <EnterpriseSettingsSection title="企业安全" labelled-by="enterprise-security-title">
          <EnterpriseSettingRow
            label="全局水印"
            description="设置全局水印的开关状态及样式，配置后将同步生效于应用和知识库水印。"
          >
            <el-switch v-model="settings.watermark" aria-label="全局水印" />
            <el-button plain @click="showPlannedSetting('全局水印')">
              设置
            </el-button>
          </EnterpriseSettingRow>

          <EnterpriseSettingRow label="单点登录" stacked>
            <div class="enterprise-settings-panel__stacked-controls">
              <div>
                <strong>外部账号登录本系统</strong>
                <el-switch
                  :model-value="settings.externalAccountLogin"
                  aria-label="外部账号登录本系统"
                  @change="toggleSso"
                />
                <el-button
                  v-if="settings.externalAccountLogin"
                  plain
                  @click="ssoDialogVisible = true"
                >
                  设置
                </el-button>
              </div>
              <div>
                <strong>本系统账号登录外部</strong>
                <el-button plain @click="showPlannedSetting('外部应用登录')">
                  前往配置
                </el-button>
              </div>
            </div>
            <template #description>
              <div class="enterprise-settings-panel__stacked-descriptions">
                <p>
                  成员可用第三方账号一键登录企业账号 URL 及发布给成员的内链。
                  <button type="button" @click="learnMore('单点登录')">
                    了解更多
                  </button>
                </p>
                <p>
                  成员可用本系统账号登录产品中心已集成的外部应用。
                  <button type="button" @click="learnMore('外部应用登录')">
                    了解更多
                  </button>
                </p>
              </div>
            </template>
          </EnterpriseSettingRow>

          <EnterpriseSettingRow
            label="AI 连接"
            beta
            description="为企业内所有成员开启 AI 连接功能，以允许 AI 工具通过个人身份调用灵衍云能力。"
          >
            <el-switch v-model="settings.aiConnection" aria-label="AI 连接" />
            <template #description>
              为企业内所有成员开启 AI 连接功能，以允许 AI 工具通过个人身份调用灵衍云能力。
              <button type="button" @click="learnMore('AI 连接')">
                了解更多
              </button>
            </template>
          </EnterpriseSettingRow>
        </EnterpriseSettingsSection>

        <EnterpriseSettingsSection title="企业文化" labelled-by="enterprise-culture-title">
          <EnterpriseSettingRow
            label="自定义登录页"
            description="自定义登录页 Logo、展示图及登录方式等，对企业账号 URL 和发布给成员的内链生效。"
          >
            <el-switch
              :model-value="settings.customLogin"
              aria-label="自定义登录页"
              @change="toggleCustomLogin"
            />
            <el-button v-if="settings.customLogin" plain @click="loginStyleDialogVisible = true">
              设置
            </el-button>
          </EnterpriseSettingRow>

          <EnterpriseSettingRow label="企业风格">
            <el-switch
              :model-value="settings.enterpriseStyle"
              aria-label="企业风格"
              @change="toggleEnterpriseStyle"
            />
            <el-popover
              v-model:visible="themePopoverVisible"
              :teleported="false"
              placement="right-start"
              :width="500"
              trigger="click"
              popper-class="enterprise-theme-popover"
            >
              <template #reference>
                <el-button v-if="settings.enterpriseStyle" plain>
                  设置
                </el-button>
              </template>
              <div class="enterprise-settings-panel__theme-picker">
                <div class="enterprise-settings-panel__theme-heading">
                  <strong>企业主题色</strong>
                  <span>对企业内所有功能模块的电脑端和移动端同时生效</span>
                </div>
                <div class="enterprise-settings-panel__theme-colors" role="radiogroup" aria-label="企业主题色">
                  <button
                    v-for="color in themeColors"
                    :key="color"
                    type="button"
                    :style="{ backgroundColor: color }"
                    :aria-label="`选择主题色 ${color}`"
                    :aria-checked="selectedTheme === color"
                    role="radio"
                    @click="selectedTheme = color"
                  >
                    <RiCheckLine v-if="selectedTheme === color" aria-hidden="true" />
                  </button>
                </div>
                <div class="enterprise-settings-panel__theme-actions">
                  <el-button @click="themePopoverVisible = false">
                    取消
                  </el-button>
                  <el-button type="primary" @click="saveTheme">
                    确定
                  </el-button>
                </div>
              </div>
            </el-popover>
            <template #description>
              自定义企业风格。
              <button type="button" @click="learnMore('企业风格')">
                了解更多
              </button>
            </template>
          </EnterpriseSettingRow>
        </EnterpriseSettingsSection>

        <EnterpriseSettingsSection title="企业协作" labelled-by="enterprise-collaboration-title">
          <EnterpriseSettingRow label="提醒屏蔽">
            <el-button plain @click="reminderDialogVisible = true">
              设置
            </el-button>
            <template #description>
              可以设置成员是否接收应用内相关提醒。
              <button type="button" @click="learnMore('提醒屏蔽')">
                了解更多
              </button>
            </template>
          </EnterpriseSettingRow>

          <EnterpriseSettingRow label="系统语言">
            <el-select v-model="settings.language" class="enterprise-settings-panel__language-select">
              <el-option label="简体中文" value="zh-CN" />
              <el-option label="繁體中文" value="zh-TW" />
              <el-option label="English" value="en-US" />
            </el-select>
            <template #description>
              设置系统内的默认显示语言。
              <button type="button" @click="learnMore('系统语言')">
                了解更多
              </button>
            </template>
          </EnterpriseSettingRow>

          <EnterpriseSettingRow label="系统时区">
            <span class="enterprise-settings-panel__timezone">{{ settings.timezone }}</span>
            <button
              class="enterprise-settings-panel__text-action"
              type="button"
              @click="timezoneDialogVisible = true"
            >
              修改
            </button>
            <template #description>
              系统内所有的时间都基于系统时区显示并存储。
              <button type="button" @click="learnMore('系统时区')">
                了解更多
              </button>
            </template>
          </EnterpriseSettingRow>

          <EnterpriseSettingRow label="AI 能力">
            <el-switch v-model="settings.aiCapability" aria-label="AI 能力" />
            <el-button plain @click="showPlannedSetting('AI 能力')">
              设置
            </el-button>
            <template #description>
              探索 AI 前沿，做智能时代先行者。
              <button type="button" @click="learnMore('AI 能力')">
                了解更多
              </button>
            </template>
          </EnterpriseSettingRow>

          <EnterpriseSettingRow label="微信服务号集成">
            <el-switch v-model="settings.wechatIntegration" aria-label="微信服务号集成" />
            <template #description>
              将系统嵌入企业微信服务号供成员使用。
              <button type="button" @click="learnMore('微信服务号集成')">
                了解更多
              </button>
            </template>
          </EnterpriseSettingRow>
        </EnterpriseSettingsSection>
      </div>
    </el-scrollbar>

    <SsoSettingsDialog v-model="ssoDialogVisible" />
    <LoginStyleDialog v-model="loginStyleDialogVisible" />
    <ReminderSuppressionDialog v-model="reminderDialogVisible" />

    <el-dialog v-model="timezoneDialogVisible" class="enterprise-timezone-dialog" width="480px">
      <template #header>
        <h2>修改系统时区</h2>
      </template>
      <el-select v-model="pendingTimezone" style="width: 100%">
        <el-option label="(GMT+8:00) Asia/Shanghai" value="(GMT+8:00) Asia/Shanghai" />
        <el-option label="(GMT+8:00) Asia/Taipei" value="(GMT+8:00) Asia/Taipei" />
        <el-option label="(GMT+9:00) Asia/Tokyo" value="(GMT+9:00) Asia/Tokyo" />
      </el-select>
      <template #footer>
        <el-button @click="timezoneDialogVisible = false">
          取消
        </el-button>
        <el-button type="primary" @click="saveTimezone">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.enterprise-settings-panel {
  --enterprise-accent: #0bb6aa;
  --el-color-primary: var(--enterprise-accent);
  height: 100%;
  min-height: 0;
  color: #202939;

  &__scrollbar {
    height: 100%;
  }

  &__content {
    min-width: 860px;
    padding: 0 20px 18px;
  }

  &__stacked-controls,
  &__stacked-descriptions {
    display: grid;
    gap: 12px;
  }

  &__stacked-controls > div {
    display: flex;
    min-height: 40px;
    align-items: center;
    gap: 12px;

    strong {
      width: 180px;
      color: #3e4758;
      font-size: 14px;
      font-weight: 600;
    }
  }

  &__stacked-descriptions p {
    min-height: 40px;
    margin: 0;
    line-height: 40px;
  }

  &__language-select {
    width: 220px;
  }

  &__timezone {
    color: #344054;
    font-size: 14px;
  }

  &__text-action,
  :deep(.enterprise-setting-row__description button) {
    padding: 0;
    border: 0;
    color: var(--enterprise-accent);
    background: transparent;
    font: inherit;
    cursor: pointer;
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }

    &:focus-visible {
      outline: 2px solid var(--enterprise-accent);
      outline-offset: 2px;
    }
  }

  &__theme-picker {
    padding: 8px 6px 2px;
  }

  &__theme-heading {
    display: flex;
    margin-bottom: 18px;
    align-items: baseline;
    gap: 14px;

    strong {
      color: #202939;
      font-size: 17px;
    }

    span {
      color: #7a8495;
      font-size: 13px;
    }
  }

  &__theme-colors {
    display: flex;
    flex-wrap: wrap;
    gap: 13px;

    button {
      display: inline-flex;
      width: 32px;
      height: 32px;
      padding: 0;
      border: 2px solid transparent;
      border-radius: 50%;
      align-items: center;
      justify-content: center;
      color: #fff;
      cursor: pointer;

      &[aria-checked='true'] {
        border-color: #fff;
        box-shadow: 0 0 0 2px currentColor;
      }

      svg {
        width: 17px;
        height: 17px;
      }
    }
  }

  &__theme-actions {
    display: flex;
    margin-top: 26px;
    justify-content: flex-end;
    gap: 10px;
  }

  :deep(.el-switch.is-checked .el-switch__core) {
    border-color: var(--enterprise-accent);
    background-color: var(--enterprise-accent);
  }

  :deep(.el-button--primary) {
    border-color: var(--enterprise-accent);
    background: var(--enterprise-accent);
  }

  :deep(.el-button.is-plain) {
    border-color: var(--enterprise-accent);
    color: var(--enterprise-accent);
    background: #fff;
  }

  :deep(.el-input__wrapper),
  :deep(.el-select__wrapper) {
    min-height: 36px;
    border-radius: 6px;
  }
}

:global(.enterprise-theme-popover.el-popper) {
  --el-color-primary: #0bb6aa;
  padding: 20px;
  border: 0;
  border-radius: 10px;
  box-shadow: 0 12px 36px rgba(26, 38, 56, 0.15);
}

:global(.enterprise-timezone-dialog) {
  --el-color-primary: #0bb6aa;
  border-radius: 12px;
}

:global(.enterprise-timezone-dialog h2) {
  margin: 0;
  font-size: 20px;
}

@media (max-width: 920px) {
  .enterprise-settings-panel__content {
    min-width: 720px;
  }
}
</style>
