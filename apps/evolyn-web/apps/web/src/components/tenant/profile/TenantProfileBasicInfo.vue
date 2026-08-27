<script setup lang="ts">
import { RiCheckFill, RiFileCopyFill, RiQuestionFill, RiShieldCheckFill } from '@remixicon/vue';

interface Props {
  accountMode: string;
  companyName: string;
  companyUrl: string;
  tenantIdentifier: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  copy: [value: string, label: string];
  editName: [];
  preview: [];
}>();
</script>

<template>
  <section class="tenant-profile-basic-info" aria-labelledby="tenant-profile-basic-info-title">
    <h1 id="tenant-profile-basic-info-title" class="tenant-profile-basic-info__title">基础信息</h1>

    <dl class="tenant-profile-basic-info__list">
      <div class="tenant-profile-basic-info__row">
        <dt class="tenant-profile-basic-info__label">企业名称</dt>
        <dd class="tenant-profile-basic-info__value tenant-profile-basic-info__value--name">
          <span>{{ props.companyName }}</span>
          <button
            class="tenant-profile-basic-info__text-action"
            type="button"
            @click="emit('editName')"
          >
            修改
          </button>
        </dd>
      </div>

      <div class="tenant-profile-basic-info__row">
        <dt class="tenant-profile-basic-info__label">账号模式</dt>
        <dd class="tenant-profile-basic-info__value tenant-profile-basic-info__mode-row">
          <el-tag effect="plain" type="info">{{ props.accountMode }}</el-tag>
          <span class="tenant-profile-basic-info__hint">如需绑定第三方平台，</span>
          <button class="tenant-profile-basic-info__text-action" type="button">点此咨询</button>
        </dd>
      </div>

      <div class="tenant-profile-basic-info__row tenant-profile-basic-info__row--identifier">
        <dt class="tenant-profile-basic-info__label">租户 ID</dt>
        <dd class="tenant-profile-basic-info__value">
          <div class="tenant-profile-basic-info__identifier">
            <span>{{ props.tenantIdentifier }}</span>
            <el-tooltip content="复制租户 ID" placement="top">
              <button
                class="tenant-profile-basic-info__icon-action"
                type="button"
                aria-label="复制租户 ID"
                @click="emit('copy', props.tenantIdentifier, '租户 ID')"
              >
                <RiFileCopyFill />
              </button>
            </el-tooltip>
          </div>
        </dd>
      </div>

      <div class="tenant-profile-basic-info__row">
        <dt class="tenant-profile-basic-info__label">企业账号 URL</dt>
        <dd class="tenant-profile-basic-info__value tenant-profile-basic-info__url-row">
          <span class="tenant-profile-basic-info__url">{{ props.companyUrl }}</span>
          <button
            class="tenant-profile-basic-info__text-action"
            type="button"
            @click="emit('copy', props.companyUrl, '企业账号 URL')"
          >
            复制
          </button>
          <button
            class="tenant-profile-basic-info__text-action"
            type="button"
            @click="emit('preview')"
          >
            预览
          </button>
        </dd>
      </div>

      <div class="tenant-profile-basic-info__row">
        <dt class="tenant-profile-basic-info__label">身份认证</dt>
        <dd class="tenant-profile-basic-info__value tenant-profile-basic-info__verification">
          <div class="tenant-profile-basic-info__verification-item">
            <span class="tenant-profile-basic-info__verification-badge">
              <RiQuestionFill /> 企业认证
            </span>
            <span class="tenant-profile-basic-info__hint">尚未企业认证</span>
            <button class="tenant-profile-basic-info__text-action" type="button">现在认证</button>
          </div>
          <div class="tenant-profile-basic-info__verification-item">
            <span
              class="tenant-profile-basic-info__verification-badge tenant-profile-basic-info__verification-badge--verified"
            >
              <RiCheckFill /> 个人认证
            </span>
            <span class="tenant-profile-basic-info__hint">已完成实名认证</span>
            <button class="tenant-profile-basic-info__text-action" type="button">重新认证</button>
          </div>
        </dd>
      </div>
    </dl>

    <div class="tenant-profile-basic-info__security-note">
      <RiShieldCheckFill aria-hidden="true" />
      <span>企业信息仅对本企业管理员可见</span>
    </div>
  </section>
</template>

<style scoped lang="scss">
.tenant-profile-basic-info {
  &__title {
    position: relative;
    margin: 0;
    padding-left: var(--el-space-xl);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-extra-large);
    font-weight: 650;
    line-height: 28px;

    &::before {
      position: absolute;
      top: 4px;
      bottom: 4px;
      left: 0;
      width: 4px;
      border-radius: var(--el-border-radius-half);
      background: var(--el-color-primary);
      content: '';
    }
  }

  &__list {
    display: grid;
    margin: var(--el-space-5xl) 0 0;
    gap: var(--el-space-3xl);
  }

  &__row {
    display: grid;
    min-height: 32px;
    grid-template-columns: 220px minmax(0, 1fr);
    align-items: center;
  }

  &__row--identifier {
    align-items: start;
  }

  &__label {
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    line-height: 24px;
  }

  &__value {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: var(--el-space-xl);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    line-height: 24px;
  }

  &__value--name {
    gap: var(--el-space-xl);
  }

  &__mode-row {
    gap: var(--el-space-lg);
  }

  &__hint,
  &__url {
    color: var(--el-text-color-secondary);
  }

  &__url-row {
    gap: var(--el-space-xl);
  }

  &__url {
    overflow: hidden;
    flex: 0 1 auto;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__text-action,
  &__icon-action {
    display: inline-flex;
    border: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
    line-height: inherit;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    &:hover {
      color: var(--el-color-primary-light-3);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__text-action {
    padding: var(--el-space-xs) var(--el-space-xs);
    border-radius: var(--el-border-radius-base);
    white-space: nowrap;
  }

  &__identifier {
    display: flex;
    box-sizing: border-box;
    width: min(100%, 680px);
    min-height: 76px;
    padding: 0 var(--el-space-xl) 0 var(--el-space-3xl);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    justify-content: space-between;
    gap: var(--el-space-xl);
    color: var(--el-text-color-regular);
  }

  &__icon-action {
    width: 32px;
    height: 32px;
    flex: 0 0 32px;
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-regular);

    svg {
      width: 19px;
      height: 19px;
    }
  }

  &__verification {
    display: grid;
    align-items: start;
    gap: var(--el-space-lg);
  }

  &__verification-item {
    display: flex;
    min-height: 32px;
    align-items: center;
    gap: var(--el-space-lg);
  }

  &__verification-badge {
    display: inline-flex;
    min-width: 136px;
    box-sizing: border-box;
    padding: var(--el-space-xs) var(--el-space-lg);
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    justify-content: center;
    gap: var(--el-space-sm);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);

    svg {
      width: 17px;
      height: 17px;
      color: var(--el-text-color-secondary);
    }
  }

  &__verification-badge--verified {
    border-color: var(--el-color-success);
    color: var(--el-color-success);
    background: var(--el-color-success-light-9);

    svg {
      color: var(--el-color-success);
    }
  }

  &__security-note {
    display: inline-flex;
    margin-top: var(--el-space-3xl);
    align-items: center;
    gap: var(--el-space-sm);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
    line-height: 20px;

    svg {
      width: 16px;
      height: 16px;
      color: var(--el-color-primary);
    }
  }
}

@media (max-width: 840px) {
  .tenant-profile-basic-info {
    &__row {
      gap: var(--el-space-md);
      grid-template-columns: 1fr;
    }

    &__list {
      gap: var(--el-space-2xl);
    }

    &__url-row {
      flex-wrap: wrap;
    }

    &__verification-item {
      flex-wrap: wrap;
    }
  }
}
</style>
