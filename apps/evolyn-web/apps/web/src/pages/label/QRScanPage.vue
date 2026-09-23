<script setup lang="ts">
import { RiQrScan2Line, RiRefreshLine } from '@remixicon/vue';
import { computed, onMounted, shallowRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { resolveLabelQRToken } from '~/api/label';

defineOptions({ name: 'LabelQRScanPage' });

interface QRRouteParams {
  token?: string | string[];
}

const route = useRoute();
const router = useRouter();
const status = shallowRef<'loading' | 'error'>('loading');
const errorMessage = shallowRef('');
const token = computed(() => {
  const value = (route.params as QRRouteParams).token;
  return Array.isArray(value) ? (value[0] ?? '') : (value ?? '');
});

async function resolveTarget(): Promise<void> {
  status.value = 'loading';
  errorMessage.value = '';
  try {
    const target = await resolveLabelQRToken(token.value);
    await router.replace({
      name: 'form-data',
      params: { appCode: target.appCode, formCode: target.formCode },
      query: { recordId: target.recordId, source: 'qrcode' },
    });
  } catch {
    status.value = 'error';
    errorMessage.value = '二维码已失效，或当前账号没有查看这条数据的权限。';
  }
}

onMounted(resolveTarget);
</script>

<template>
  <main class="qr-scan-page">
    <section class="qr-scan-card" aria-live="polite">
      <span class="qr-scan-card__icon" aria-hidden="true">
        <RiQrScan2Line />
      </span>
      <template v-if="status === 'loading'">
        <h1>正在定位数据</h1>
        <p>
          正在校验二维码和访问权限，请稍候…
        </p>
      </template>
      <template v-else>
        <h1>无法打开二维码</h1>
        <p role="alert">
          {{ errorMessage }}
        </p>
        <el-button type="primary" @click="resolveTarget">
          <RiRefreshLine />
          重新尝试
        </el-button>
      </template>
    </section>
  </main>
</template>

<style scoped lang="scss">
.qr-scan-page {
  display: grid;
  min-height: 100vh;
  padding: var(--el-space-3xl);
  place-items: center;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-page);
}

.qr-scan-card {
  display: flex;
  width: min(100%, 420px);
  padding: var(--el-space-4xl);
  align-items: center;
  flex-direction: column;
  text-align: center;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);

  h1 {
    margin: var(--el-space-xl) 0 var(--el-space-sm);
    font-size: var(--el-font-size-extra-large);
  }

  p {
    margin: 0 0 var(--el-space-xl);
    color: var(--el-text-color-secondary);
    line-height: var(--el-font-line-height-primary);
  }

  &__icon {
    display: grid;
    width: var(--el-component-size-large);
    height: var(--el-component-size-large);
    color: var(--el-color-primary);
    place-items: center;
    background: var(--el-color-primary-light-9);
    border-radius: var(--el-border-radius-circle);

    svg {
      width: var(--el-font-size-extra-large);
      height: var(--el-font-size-extra-large);
    }
  }
}
</style>
