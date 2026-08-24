<script setup lang="ts">
import { RiCoinsFill, RiQuestionFill } from '@remixicon/vue';

const props = defineProps<{
  balance: string;
  expiryDate: string;
  versionName: string;
}>();

const emit = defineEmits<{
  action: [
    action: 'consult' | 'certificate' | 'recharge' | 'payment' | 'consumption' | 'usage-log',
  ];
}>();
</script>

<template>
  <section class="edition-overview" aria-label="当前套餐和云币">
    <div class="edition-overview__column">
      <h1 class="edition-overview__section-title">当前版本</h1>
      <article class="edition-overview__trial-card">
        <div class="edition-overview__trial-content">
          <div class="edition-overview__trial-heading">
            <strong>{{ props.versionName }}</strong>
            <span>即日起-{{ props.expiryDate }}</span>
          </div>
          <p>
            版本到期后将变为免费版，免费版不包含高级功能，为确保试用到期后正常使用，请根据使用情况升级合适的版本。
          </p>
          <div class="edition-overview__actions">
            <el-button type="primary" @click="emit('action', 'consult')">咨询购买</el-button>
            <el-button @click="emit('action', 'certificate')">上传凭证</el-button>
          </div>
        </div>
        <div class="edition-overview__trial-illustration" aria-hidden="true">
          <span class="edition-overview__illustration-base" />
          <span class="edition-overview__illustration-cube">
            <span class="edition-overview__illustration-gem">◆</span>
          </span>
        </div>
      </article>
    </div>

    <div class="edition-overview__column">
      <h2 class="edition-overview__section-title">
        云币
        <el-tooltip content="云币可用于支付超出套餐后的资源用量" placement="top">
          <RiQuestionFill aria-label="云币说明" />
        </el-tooltip>
      </h2>
      <article class="edition-overview__coin-card">
        <div>
          <p class="edition-overview__coin-balance">
            {{ props.balance }}
            <span>（1 云币 = ￥1）</span>
          </p>
          <div class="edition-overview__actions">
            <el-button type="primary" @click="emit('action', 'recharge')">充值</el-button>
            <el-button @click="emit('action', 'payment')">支付设置</el-button>
            <el-button @click="emit('action', 'consumption')">消耗统计</el-button>
            <el-button @click="emit('action', 'usage-log')">使用日志</el-button>
          </div>
        </div>
        <div class="edition-overview__coin-illustration" aria-hidden="true">
          <RiCoinsFill />
        </div>
      </article>
    </div>
  </section>
</template>

<style scoped lang="scss">
.edition-overview {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;

  &__column {
    min-width: 0;
  }

  &__section-title {
    display: flex;
    margin: 0 0 14px;
    align-items: center;
    gap: 8px;
    color: #1f2937;
    font-size: 18px;
    font-weight: 650;
    line-height: 28px;

    &::before {
      width: 5px;
      height: 20px;
      border-radius: 4px;
      background: var(--el-color-primary);
      content: '';
    }

    svg {
      width: 18px;
      height: 18px;
      color: #a9b1bd;
    }
  }

  &__trial-card,
  &__coin-card {
    position: relative;
    display: flex;
    min-height: 202px;
    box-sizing: border-box;
    overflow: hidden;
    border: 1px solid #e8eff8;
    border-radius: 14px;
  }

  &__trial-card {
    padding: 30px 184px 28px 28px;
    background: linear-gradient(120deg, #f1f7ff 0%, #eaf4ff 100%);
  }

  &__trial-content {
    position: relative;
    z-index: 1;
  }

  &__trial-heading {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 16px;
    white-space: nowrap;

    strong {
      color: #152237;
      font-size: 30px;
      font-weight: 650;
      letter-spacing: 0.01em;
      line-height: 40px;
    }

    span {
      color: #8490a2;
      font-size: 16px;
      line-height: 24px;
    }
  }

  &__trial-content p {
    margin: 4px 0 14px;
    color: #778397;
    font-size: 15px;
    line-height: 24px;
  }

  &__actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;

    :deep(.el-button) {
      height: 36px;
      margin: 0;
      padding: 0 16px;
      border-color: #d9e1eb;
      color: #384152;
      font-size: 15px;
    }

    :deep(.el-button--primary) {
      border-color: var(--el-color-primary);
      color: #fff;
    }
  }

  &__trial-illustration {
    position: absolute;
    right: 2px;
    bottom: -2px;
    width: 170px;
    height: 170px;
  }

  &__illustration-base {
    position: absolute;
    right: 0;
    bottom: 0;
    width: 148px;
    height: 80px;
    background: linear-gradient(135deg, #e0f0ff, #cfe9ff);
    clip-path: polygon(50% 0, 100% 35%, 50% 70%, 0 35%);

    &::after {
      position: absolute;
      top: 28px;
      left: 0;
      width: 100%;
      height: 100%;
      background: #d7edff;
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
    background: linear-gradient(135deg, #e9f7ff, #c3e4ff);
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
    background: linear-gradient(135deg, #00c6ee, #428cff);
    box-shadow: 0 5px 10px rgb(37 139 229 / 30%);
    font-size: 22px;

    &::first-letter {
      transform: rotate(-45deg);
    }
  }

  &__coin-card {
    justify-content: space-between;
    padding: 30px 28px;
    border-color: #fbe7bf;
    background: linear-gradient(120deg, #fffaf0 0%, #fff5df 100%);
  }

  &__coin-balance {
    margin: 0 0 68px;
    color: #152237;
    font-size: 30px;
    font-weight: 650;
    line-height: 40px;

    span {
      margin-left: 8px;
      color: #788395;
      font-size: 16px;
      font-weight: 400;
    }
  }

  &__coin-illustration {
    position: absolute;
    right: 30px;
    bottom: 27px;
    display: grid;
    width: 96px;
    height: 116px;
    place-items: center;
    color: #f5a51d;
    background: linear-gradient(145deg, #fff1c9, #ffe0a4);
    clip-path: polygon(50% 0, 100% 27%, 100% 73%, 50% 100%, 0 73%, 0 27%);

    svg {
      width: 50px;
      height: 50px;
      filter: drop-shadow(0 4px 5px rgb(225 145 15 / 24%));
    }
  }
}

@media (max-width: 1180px) {
  .edition-overview {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .edition-overview {
    gap: 18px;

    &__trial-card {
      padding: 24px;
    }

    &__trial-heading {
      align-items: flex-start;
      flex-direction: column;
      gap: 1px;
    }

    &__trial-illustration,
    &__coin-illustration {
      opacity: 0.3;
    }

    &__coin-balance {
      margin-bottom: 34px;
    }
  }
}
</style>
