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
    padding-left: 16px;
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 650;
    line-height: 28px;

    &::before {
      position: absolute;
      top: 4px;
      bottom: 4px;
      left: 0;
      width: 4px;
      border-radius: 999px;
      background: var(--el-color-primary);
      content: '';
    }
  }

  &__list {
    display: grid;
    margin: 40px 0 0;
    gap: 28px;
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
    font-size: 16px;
    font-weight: 600;
    line-height: 24px;
  }

  &__value {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 18px;
    color: var(--el-text-color-primary);
    font-size: 16px;
    line-height: 24px;
  }

  &__value--name {
    gap: 16px;
  }

  &__mode-row {
    gap: 14px;
  }

  &__hint,
  &__url {
    color: var(--el-text-color-secondary);
  }

  &__url-row {
    gap: 18px;
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
    padding: 2px 4px;
    border-radius: 4px;
    white-space: nowrap;
  }

  &__identifier {
    display: flex;
    box-sizing: border-box;
    width: min(100%, 680px);
    min-height: 76px;
    padding: 0 18px 0 26px;
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    align-items: center;
    justify-content: space-between;
    gap: 18px;
    color: var(--el-text-color-regular);
  }

  &__icon-action {
    width: 32px;
    height: 32px;
    flex: 0 0 32px;
    border-radius: 6px;
    color: var(--el-text-color-regular);

    svg {
      width: 19px;
      height: 19px;
    }
  }

  &__verification {
    display: grid;
    align-items: start;
    gap: 14px;
  }

  &__verification-item {
    display: flex;
    min-height: 32px;
    align-items: center;
    gap: 14px;
  }

  &__verification-badge {
    display: inline-flex;
    min-width: 136px;
    box-sizing: border-box;
    padding: 5px 14px;
    border: 1px dashed var(--el-border-color);
    border-radius: 6px;
    align-items: center;
    justify-content: center;
    gap: 6px;
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
    margin-top: 28px;
    align-items: center;
    gap: 6px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
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
      gap: 10px;
      grid-template-columns: 1fr;
    }

    &__list {
      gap: 22px;
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
