<script setup lang="ts">
import type { DeepReadonly } from 'vue';
import type { UserInfoResult } from '~/types';
import { RiErrorWarningFill, RiInformationFill, RiLinksFill } from '@remixicon/vue';

defineOptions({ name: 'AccountSecurityPanel' });

defineProps<{
  userInfo: DeepReadonly<UserInfoResult> | null;
}>();
</script>

<template>
  <section class="account-security" aria-label="账号安全">
    <div class="account-security__row">
      <div>
        <strong>登录二次验证</strong>
        <el-tooltip content="该策略需由后端安全服务启用后才可配置" placement="top">
          <el-icon class="account-security__help"><RiInformationFill /></el-icon>
        </el-tooltip>
      </div>
      <el-switch disabled aria-label="登录二次验证暂未开放" />
    </div>
    <div class="account-security__row">
      <div>
        <strong>禁止同时登录</strong>
        <el-tooltip content="该策略需由后端会话服务启用后才可配置" placement="top">
          <el-icon class="account-security__help"><RiInformationFill /></el-icon>
        </el-tooltip>
      </div>
      <el-switch disabled aria-label="禁止同时登录暂未开放" />
    </div>

    <div class="account-security__binding">
      <strong>账号绑定</strong>
      <div class="account-security__binding-content">
        <template v-if="userInfo?.account.authInfos.length">
          <div
            v-for="authInfo in userInfo.account.authInfos"
            :key="authInfo.id"
            class="account-security__provider"
          >
            <el-icon><RiLinksFill /></el-icon>
            <span>{{ authInfo.authType }}</span>
            <el-tag size="small" type="success" effect="plain">已绑定</el-tag>
          </div>
        </template>
        <span v-else class="account-security__muted">暂未绑定第三方账号</span>
        <el-tooltip content="第三方账号绑定能力正在建设中" placement="top">
          <el-button link type="primary" disabled>管理绑定</el-button>
        </el-tooltip>
      </div>
    </div>

    <el-divider />

    <div class="account-security__danger">
      <strong>
        <el-icon><RiErrorWarningFill /></el-icon>
        账号注销
      </strong>
      <span>
        <el-button link type="danger">注销</el-button>
      </span>
    </div>
  </section>
</template>

<style scoped lang="scss">
.account-security {
  &__row,
  &__binding,
  &__danger {
    display: grid;
    grid-template-columns: 162px minmax(0, 1fr);
    align-items: center;
    min-height: 54px;
  }

  &__row > div,
  &__binding > strong,
  &__danger > strong {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__help {
    margin-left: 4px;
    vertical-align: -2px;
    color: var(--el-text-color-secondary);
    cursor: help;
  }

  &__binding {
    align-items: start;
    padding: 12px 0;
  }

  &__binding-content {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    min-height: 30px;
    gap: 12px;
  }

  &__provider {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--el-text-color-regular);
  }

  &__muted,
  &__danger > span {
    color: var(--el-text-color-secondary);
  }

  &__danger > strong {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
}
</style>
