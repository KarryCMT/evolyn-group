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
    padding: 8px 0 38px;
  }
  &__switch-row > div {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 24px;
  }
  &__switch-row h2,
  &__address h3 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 600;
    line-height: 30px;
  }
  &__switch-row h2 span {
    margin-left: 14px;
    color: var(--el-text-color-secondary);
    font-size: 16px;
    font-weight: 400;
  }
  &__divider {
    height: 1px;
    background: var(--el-border-color-lighter);
  }
  &__address {
    padding-top: 40px;
  }
  &__address h3 {
    margin-bottom: 18px;
  }
  &__address :deep(.el-input-group__append) {
    padding: 0;
    background: var(--el-bg-color);
  }
  &__address :deep(.el-input-group__append button) {
    display: inline-flex;
    height: 42px;
    padding: 0 18px;
    border: 0;
    border-left: 1px solid var(--el-border-color);
    align-items: center;
    gap: 6px;
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
    padding: 24px 28px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 12px;
    flex-direction: column;
    align-items: center;
    background: var(--el-bg-color-overlay);
    box-shadow: var(--el-box-shadow-light);
  }
  &__qr-card p {
    margin: 0 0 14px;
    color: var(--el-text-color-regular);
    font-size: 16px;
  }
  &__qr-card img {
    width: 220px;
    height: 220px;
  }
  &__qr-card button {
    display: inline-flex;
    width: 100%;
    height: 40px;
    margin-top: 15px;
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    align-items: center;
    justify-content: center;
    gap: 8px;
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
    margin: 28px 0 0 auto;
  }
}
</style>
