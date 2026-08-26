<script setup lang="ts">
import { RiErrorWarningFill, RiInformationFill } from '@remixicon/vue';
import type { EditionQuotaCard } from './types';

const props = defineProps<{
  cards: EditionQuotaCard[];
}>();

const emit = defineEmits<{
  detail: [card: EditionQuotaCard];
  upgrade: [];
}>();
</script>

<template>
  <section class="edition-quotas" aria-labelledby="edition-quota-title">
    <h2 id="edition-quota-title" class="edition-quotas__section-title">系统容量</h2>
    <div class="edition-quotas__grid">
      <article v-for="card in props.cards" :key="card.id" class="edition-quotas__card">
        <header class="edition-quotas__card-header">
          <div class="edition-quotas__title-wrap">
            <span class="edition-quotas__icon" :class="`edition-quotas__icon--${card.tone}`">
              <component :is="card.icon" aria-hidden="true" />
            </span>
            <h3>{{ card.title }}</h3>
          </div>
          <span v-if="card.warning" class="edition-quotas__warning">
            {{ card.warning }}
            <RiErrorWarningFill aria-hidden="true" />
          </span>
        </header>

        <!-- 待计量资源不渲染进度条（无真实已用值，禁止伪 0 展示） -->
        <div
          v-if="card.meteringStatus === 'ready'"
          class="edition-quotas__progress"
          role="progressbar"
          :aria-valuenow="card.progress"
          aria-valuemin="0"
          aria-valuemax="100"
        >
          <i :style="{ width: `${card.progress}%` }" />
        </div>
        <div v-else class="edition-quotas__progress edition-quotas__progress--pending" />
        <div class="edition-quotas__metrics">
          <span>{{ card.usageLabel }}</span>
          <span>
            {{ card.limitLabel }}
            <el-tooltip v-if="card.note" :content="card.note" placement="top">
              <RiInformationFill aria-label="容量说明" />
            </el-tooltip>
          </span>
        </div>
        <p v-if="card.note" class="edition-quotas__note">{{ card.note }}</p>
        <footer class="edition-quotas__footer">
          <button type="button" @click="emit('detail', card)">查看详情</button>
        </footer>
      </article>
    </div>
  </section>
</template>

<style scoped lang="scss">
.edition-quotas {
  margin-top: 38px;

  &__section-title {
    display: flex;
    margin: 0 0 16px;
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

  &__grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 20px;
  }

  &__card {
    display: flex;
    min-height: 220px;
    box-sizing: border-box;
    flex-direction: column;
    padding: 28px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 14px;
    background: var(--el-bg-color);
    box-shadow: 0 1px 3px rgb(36 51 73 / 5%);
  }

  &__card-header,
  &__title-wrap,
  &__warning,
  &__metrics,
  &__footer {
    display: flex;
    align-items: center;
  }

  &__card-header {
    min-height: 34px;
    justify-content: space-between;
    gap: 10px;
  }

  &__title-wrap {
    min-width: 0;
    gap: 12px;

    h3 {
      overflow: hidden;
      margin: 0;
      color: var(--el-text-color-primary);
      font-size: 18px;
      font-weight: 650;
      line-height: 28px;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  &__icon {
    display: grid;
    width: 34px;
    height: 34px;
    flex: 0 0 34px;
    place-items: center;
    border-radius: 9px;

    svg {
      width: 20px;
      height: 20px;
    }

    &--blue {
      color: #5b7ffa;
      background: #e3e9ff;
    }

    &--cyan {
      color: #11b6d4;
      background: #dff5fa;
    }

    &--green {
      color: #31ba69;
      background: #ddf7e3;
    }

    &--orange {
      color: #f49343;
      background: #ffead3;
    }

    &--purple {
      color: #a558df;
      background: #f4ddff;
    }
  }

  &__warning {
    flex: 0 0 auto;
    gap: 4px;
    color: #f1a11d;
    font-size: 13px;
    line-height: 20px;

    svg {
      width: 16px;
      height: 16px;
    }
  }

  &__progress {
    height: 12px;
    margin-top: 20px;
    overflow: hidden;
    border-radius: 10px;
    background: var(--el-fill-color);

    i {
      display: block;
      height: 100%;
      min-width: 3px;
      border-radius: inherit;
      background: var(--el-color-primary);
      transition: width 0.24s ease;
    }

    // 待计量占位：仅保留轨道撑住布局，不渲染任何进度比例
    &--pending i {
      display: none;
    }
  }

  &__metrics {
    margin-top: 12px;
    justify-content: space-between;
    gap: 10px;
    color: var(--el-text-color-regular);
    font-size: 15px;
    line-height: 24px;

    span:last-child {
      display: inline-flex;
      align-items: center;
      gap: 3px;
      text-align: right;
    }

    svg {
      width: 16px;
      height: 16px;
      color: var(--el-text-color-placeholder);
    }
  }

  &__note {
    min-height: 20px;
    margin: 8px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }

  &__footer {
    min-height: 24px;
    margin-top: auto;
    justify-content: flex-end;
    gap: 24px;

    button {
      padding: 4px 0;
      border: 0;
      border-radius: 5px;
      color: var(--el-color-primary);
      background: transparent;
      font-size: 15px;
      line-height: 20px;
      cursor: pointer;
      transition: background-color 0.18s ease;

      &:hover {
        background: var(--el-color-primary-light-9);
      }

      &:focus-visible {
        outline: 2px solid var(--el-color-primary);
        outline-offset: 2px;
      }
    }
  }
}

@media (max-width: 1400px) {
  .edition-quotas__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .edition-quotas {
    margin-top: 28px;

    &__grid {
      grid-template-columns: 1fr;
    }

    &__card {
      min-height: 202px;
      padding: 22px;
    }

    &__warning {
      display: none;
    }
  }
}
</style>
