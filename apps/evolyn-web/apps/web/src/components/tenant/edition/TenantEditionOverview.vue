<script setup lang="ts">
import { computed } from 'vue';
import type { EditionSubscription } from '~/types';

const props = defineProps<{
  subscription: EditionSubscription;
}>();

const emit = defineEmits<{
  consult: [];
}>();

/** 状态标签文案：到期为读时投影，待确认为存量试用补录态 */
const statusText = computed<string>(() => {
  switch (props.subscription.status) {
    case 'expired':
      return '已过期';
    case 'legacy_pending_review':
      return '有效期待确认';
    default:
      return '使用中';
  }
});

const statusTone = computed<'success' | 'warning' | 'info'>(() => {
  switch (props.subscription.status) {
    case 'expired':
      return 'warning';
    case 'legacy_pending_review':
      return 'info';
    default:
      return 'success';
  }
});

/** 有效期展示：长期有效不显示到期日（设计 4.3.1） */
const validityText = computed<string>(() => {
  const { startsAt, endsAt } = props.subscription;
  const start = startsAt?.slice(0, 10) ?? '';
  if (props.subscription.status === 'legacy_pending_review') {
    return '有效期待运营确认';
  }
  if (!endsAt) {
    return start ? `${start} 起 · 长期有效` : '长期有效';
  }
  return `${start} - ${endsAt.slice(0, 10)}`;
});

/** 到期处理说明：仅会降级的订阅提示升级，其余说明当前权益来源 */
const expiresHint = computed<string>(() => {
  if (props.subscription.status === 'expired') {
    return '订阅已到期，当前已按免费版提供权益；如需继续使用高级能力，请升级版本。';
  }
  if (props.subscription.expiresAction === 'downgrade_to_free') {
    return '版本到期后将降级为免费版，免费版不包含高级功能，请根据使用情况提前升级合适的版本。';
  }
  return '当前版本权益以套餐定义为准，成员、应用与附件存储用量可在下方实时查看。';
});
</script>

<template>
  <section class="edition-overview" aria-label="当前版本">
    <h1 class="edition-overview__section-title">当前版本</h1>
    <article class="edition-overview__card">
      <div class="edition-overview__content">
        <div class="edition-overview__heading">
          <strong>{{ props.subscription.planName }}</strong>
          <span class="edition-overview__validity">
            <el-tag :type="statusTone" size="small" effect="light">{{ statusText }}</el-tag>
            {{ validityText }}
          </span>
        </div>
        <p>{{ expiresHint }}</p>
        <div class="edition-overview__actions">
          <el-button type="primary" @click="emit('consult')">咨询升级</el-button>
        </div>
      </div>
      <div class="edition-overview__illustration" aria-hidden="true">
        <span class="edition-overview__illustration-base" />
        <span class="edition-overview__illustration-cube">
          <span class="edition-overview__illustration-gem">◆</span>
        </span>
      </div>
    </article>
  </section>
</template>

<style scoped lang="scss">
.edition-overview {
  &__section-title {
    display: flex;
    margin: 0 0 14px;
    align-items: center;
    color: var(--el-text-color-primary);
    font-size: 18px;
    font-weight: 650;
    line-height: 28px;

    &::before {
      width: 5px;
      height: 20px;
      margin-right: 10px;
      border-radius: 4px;
      background: var(--el-color-primary);
      content: '';
    }
  }

  &__card {
    position: relative;
    display: flex;
    min-height: 170px;
    box-sizing: border-box;
    overflow: hidden;
    padding: 30px 184px 28px 28px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 14px;
    background: linear-gradient(
      120deg,
      var(--el-color-primary-light-9) 0%,
      var(--el-color-primary-light-8) 100%
    );
  }

  &__content {
    position: relative;
    z-index: 1;
  }

  &__heading {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 16px;
    white-space: nowrap;

    strong {
      color: var(--el-text-color-primary);
      font-size: 30px;
      font-weight: 650;
      letter-spacing: 0.01em;
      line-height: 40px;
    }
  }

  &__validity {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    color: var(--el-text-color-secondary);
    font-size: 16px;
    line-height: 24px;
  }

  &__content p {
    margin: 10px 0 16px;
    max-width: 640px;
    color: var(--el-text-color-regular);
    font-size: 15px;
    line-height: 24px;
    white-space: normal;
  }

  &__actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;

    :deep(.el-button) {
      height: 36px;
      margin: 0;
      padding: 0 16px;
    }
  }

  &__illustration {
    position: absolute;
    right: 2px;
    bottom: -2px;
    width: 170px;
    height: 170px;
    pointer-events: none;
  }

  &__illustration-base {
    position: absolute;
    right: 0;
    bottom: 0;
    width: 148px;
    height: 80px;
    background: linear-gradient(
      135deg,
      var(--el-color-primary-light-7),
      var(--el-color-primary-light-5)
    );
    clip-path: polygon(50% 0, 100% 35%, 50% 70%, 0 35%);

    &::after {
      position: absolute;
      top: 28px;
      left: 0;
      width: 100%;
      height: 100%;
      background: var(--el-color-primary-light-6);
      clip-path: polygon(0 0, 50% 42%, 100% 0, 100% 52%, 50% 95%, 0 52%);
      content: '';
    }
  }

  &__illustration-cube {
    position: absolute;
    top: 28px;
    right: 35px;
    display: grid;
    width: 94px;
    height: 94px;
    place-items: center;
    background: linear-gradient(
      135deg,
      var(--el-color-primary-light-8),
      var(--el-color-primary-light-4)
    );
    clip-path: polygon(50% 0, 100% 29%, 100% 72%, 50% 100%, 0 72%, 0 29%);
    filter: drop-shadow(0 10px 16px rgb(66 155 233 / 16%));
  }

  &__illustration-gem {
    display: grid;
    width: 42px;
    height: 34px;
    transform: rotate(45deg);
    place-items: center;
    border-radius: 8px;
    color: #fff;
    background: linear-gradient(135deg, var(--el-color-primary-light-3), var(--el-color-primary));
    box-shadow: 0 5px 10px rgb(37 139 229 / 30%);
    font-size: 22px;

    &::first-letter {
      transform: rotate(-45deg);
    }
  }
}

@media (max-width: 640px) {
  .edition-overview {
    &__card {
      padding: 24px;
    }

    &__heading {
      align-items: flex-start;
      flex-direction: column;
      gap: 6px;
    }

    &__illustration {
      opacity: 0.3;
    }
  }
}
</style>
