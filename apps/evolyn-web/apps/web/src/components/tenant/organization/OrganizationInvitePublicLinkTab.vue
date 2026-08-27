<script setup lang="ts">
import QRCode from 'qrcode';
import { RiDownload2Fill, RiFileCopyFill, RiQrCodeFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';

const props = defineProps<{
  enabled: boolean;
  loading: boolean;
  invitationUrl: string;
}>();

const emit = defineEmits<{
  'update:enabled': [enabled: boolean];
}>();

const qrDataUrl = shallowRef('');
const qrVisible = shallowRef(false);
const canShare = computed(() => props.enabled && Boolean(props.invitationUrl));

async function generateQRCode() {
  if (!canShare.value) return;
  qrDataUrl.value = await QRCode.toDataURL(props.invitationUrl, {
    width: 244,
    margin: 1,
    color: { dark: '#111827', light: '#00000000' },
  });
}

async function copyInvitationUrl() {
  if (!canShare.value) return;
  try {
    await navigator.clipboard.writeText(props.invitationUrl);
    ElMessage.success('邀请链接已复制');
  } catch {
    ElMessage.warning('复制失败，请手动复制链接');
  }
}

async function openQRCode() {
  if (!canShare.value) return;
  await generateQRCode();
  qrVisible.value = true;
}

function downloadQRCode() {
  if (!qrDataUrl.value) return;
  const link = document.createElement('a');
  link.href = qrDataUrl.value;
  link.download = '成员公开邀请二维码.png';
  link.click();
}

watch(
  () => props.invitationUrl,
  () => {
    qrDataUrl.value = '';
    qrVisible.value = false;
  },
);
</script>

<template>
  <section class="organization-invite-public" aria-label="公开链接邀请">
    <div class="organization-invite-public__switch-row">
      <div>
        <h2>邀请链接 <span>可通过下方链接加入你的企业。邀请链接永久有效</span></h2>
        <el-switch
          :model-value="enabled"
          :loading="loading"
          @update:model-value="emit('update:enabled', Boolean($event))"
        />
      </div>
    </div>

    <div class="organization-invite-public__divider" />

    <section class="organization-invite-public__address">
      <h3>链接地址</h3>
      <el-input :model-value="invitationUrl" readonly placeholder="开启邀请链接后生成地址">
        <template #append>
          <button type="button" :disabled="!canShare" @click="copyInvitationUrl">
            <RiFileCopyFill />复制
          </button>
          <button type="button" :disabled="!canShare" @click="openQRCode"><RiQrCodeFill /></button>
        </template>
      </el-input>
    </section>

    <aside v-if="qrVisible" class="organization-invite-public__qr-card">
      <p>扫描二维码，分享此链接</p>
      <img :src="qrDataUrl" alt="成员公开邀请二维码" />
      <button type="button" @click="downloadQRCode"><RiDownload2Fill />下载</button>
    </aside>
  </section>
</template>

<style scoped lang="scss">
.organization-invite-public {
  position: relative;
  min-height: 320px;

  &__switch-row {
    padding: var(--el-space-md) 0 var(--el-space-5xl);
  }
  &__switch-row > div {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--el-space-3xl);
  }
  &__switch-row h2,
  &__address h3 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    line-height: 30px;
  }
  &__switch-row h2 span {
    margin-left: var(--el-space-lg);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
    font-weight: 400;
  }
  &__divider {
    height: 1px;
    background: var(--el-border-color-lighter);
  }
  &__address {
    padding-top: var(--el-space-5xl);
  }
  &__address h3 {
    margin-bottom: var(--el-space-xl);
  }
  &__address :deep(.el-input-group__append) {
    padding: 0;
    background: var(--el-bg-color);
  }
  &__address :deep(.el-input-group__append button) {
    display: inline-flex;
    height: 42px;
    padding: 0 var(--el-space-xl);
    border: 0;
    border-left: 1px solid var(--el-border-color);
    align-items: center;
    gap: var(--el-space-sm);
    color: var(--el-text-color-regular);
    background: transparent;
    cursor: pointer;
    font: inherit;
  }
  &__address :deep(.el-input-group__append button:hover:not(:disabled)) {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__address :deep(.el-input-group__append button:disabled) {
    cursor: not-allowed;
    color: var(--el-text-color-placeholder);
  }
  &__address :deep(.el-input-group__append svg) {
    width: 18px;
    height: 18px;
  }
  &__qr-card {
    position: absolute;
    top: 205px;
    right: -220px;
    display: flex;
    width: 288px;
    padding: var(--el-space-3xl) var(--el-space-3xl);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-large);
    flex-direction: column;
    align-items: center;
    background: var(--el-bg-color-overlay);
    box-shadow: var(--el-box-shadow-light);
  }
  &__qr-card p {
    margin: 0 0 var(--el-space-lg);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-medium);
  }
  &__qr-card img {
    width: 220px;
    height: 220px;
  }
  &__qr-card button {
    display: inline-flex;
    width: 100%;
    height: 40px;
    margin-top: var(--el-space-xl);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    justify-content: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-regular);
    background: transparent;
    cursor: pointer;
    font: inherit;
  }
  &__qr-card button:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}

@media (max-width: 1180px) {
  .organization-invite-public__qr-card {
    position: static;
    width: 288px;
    margin: var(--el-space-3xl) 0 0 auto;
  }
}
</style>
