<script setup lang="ts">
import { ElMessage } from 'element-plus';
import QRCode from 'qrcode';
import { computed, reactive, shallowRef, watch } from 'vue';
import { RiCloseFill, RiFileCopyFill, RiShieldKeyholeFill } from '@remixicon/vue';
import { confirmMyTOTP, enrollMyTOTP, reauthAccountSecurity } from '~/api/account';
import { encryptPassword } from '~/api/conf';
import type { TOTPEnrollment } from '~/types';

defineOptions({ name: 'TotpEnrollmentDialog' });

const props = defineProps<{
  modelValue: boolean;
  passwordInitialized: boolean;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  completed: [];
}>();

type EnrollmentStep = 'reauth' | 'scan' | 'recovery';

const step = shallowRef<EnrollmentStep>('reauth');
const submitting = shallowRef(false);
const enrollment = shallowRef<TOTPEnrollment | null>(null);
const qrDataURL = shallowRef('');
const recoveryCodes = shallowRef<string[]>([]);
const form = reactive({ password: '', code: '' });

const title = computed(() => {
  if (step.value === 'scan') return '绑定验证器';
  if (step.value === 'recovery') return '保存恢复码';
  return '验证身份';
});

function reset() {
  step.value = 'reauth';
  form.password = '';
  form.code = '';
  enrollment.value = null;
  // 二维码内含绑定密钥，关闭后立即从页面内存移除。
  qrDataURL.value = '';
  recoveryCodes.value = [];
}

function close() {
  reset();
  emit('update:modelValue', false);
}

async function startEnrollment() {
  if (!props.passwordInitialized) {
    ElMessage.warning('请先在基本资料中设置登录密码，再开启登录二次验证');
    return;
  }
  if (!form.password) {
    ElMessage.warning('请输入当前密码');
    return;
  }

  submitting.value = true;
  try {
    const encryptedPassword = await encryptPassword(form.password);
    const { reauthToken } = await reauthAccountSecurity({ password: encryptedPassword });
    const pendingEnrollment = await enrollMyTOTP(reauthToken);
    const dataURL = await QRCode.toDataURL(pendingEnrollment.otpauthUrl, {
      width: 220,
      margin: 1,
      errorCorrectionLevel: 'M',
    });
    enrollment.value = pendingEnrollment;
    qrDataURL.value = dataURL;
    form.password = '';
    step.value = 'scan';
  } catch {
    ElMessage.error('创建验证器绑定失败，请检查密码后重试');
  } finally {
    submitting.value = false;
  }
}

async function confirmEnrollment() {
  const code = form.code.trim();
  if (!/^\d{6}$/.test(code) || !enrollment.value) {
    ElMessage.warning('请输入验证器生成的 6 位动态码');
    return;
  }

  submitting.value = true;
  try {
    const result = await confirmMyTOTP(enrollment.value.enrollmentId, code);
    recoveryCodes.value = result.recoveryCodes;
    // 验证器导入地址完成使用后立即清除，避免在恢复码步骤继续保留密钥。
    qrDataURL.value = '';
    enrollment.value = null;
    form.code = '';
    step.value = 'recovery';
  } catch {
    ElMessage.error('动态码验证失败，请确认验证器时间后重试');
  } finally {
    submitting.value = false;
  }
}

async function copyRecoveryCodes() {
  try {
    await navigator.clipboard.writeText(recoveryCodes.value.join('\n'));
    ElMessage.success('恢复码已复制');
  } catch {
    ElMessage.warning('复制失败，请手动保存恢复码');
  }
}

function complete() {
  emit('completed');
  close();
}

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) reset();
  },
);
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    width="460px"
    class="totp-enrollment-dialog"
    :show-close="false"
    :close-on-click-modal="false"
    :close-on-press-escape="step !== 'recovery'"
    @update:model-value="close"
  >
    <template #header>
      <div class="totp-enrollment-dialog__header">
        <span>{{ title }}</span>
        <button
          v-if="step !== 'recovery'"
          type="button"
          aria-label="关闭登录二次验证设置"
          @click="close"
        >
          <RiCloseFill />
        </button>
      </div>
    </template>

    <section v-if="step === 'reauth'" class="totp-enrollment-dialog__content">
      <el-icon class="totp-enrollment-dialog__hero"><RiShieldKeyholeFill /></el-icon>
      <p>开启登录二次验证前，需要再次确认是你本人操作。</p>
      <el-form label-position="top" @submit.prevent="startEnrollment">
        <el-form-item label="当前密码">
          <el-input
            v-model="form.password"
            type="password"
            autocomplete="current-password"
            show-password
            placeholder="请输入当前密码"
          />
        </el-form-item>
      </el-form>
    </section>

    <section v-else-if="step === 'scan'" class="totp-enrollment-dialog__content">
      <p>使用验证器应用扫描二维码，然后输入显示的 6 位动态码。</p>
      <img
        v-if="qrDataURL"
        class="totp-enrollment-dialog__qr"
        :src="qrDataURL"
        alt="验证器绑定二维码"
      />
      <el-form label-position="top" @submit.prevent="confirmEnrollment">
        <el-form-item label="动态码">
          <el-input
            v-model="form.code"
            maxlength="6"
            inputmode="numeric"
            autocomplete="one-time-code"
            placeholder="请输入 6 位动态码"
          />
        </el-form-item>
      </el-form>
    </section>

    <section v-else class="totp-enrollment-dialog__content">
      <p class="totp-enrollment-dialog__warning">
        恢复码仅显示这一次。请保存至安全位置；遗失后可能无法恢复账号访问。
      </p>
      <div class="totp-enrollment-dialog__codes" aria-label="MFA 恢复码">
        <code v-for="code in recoveryCodes" :key="code">{{ code }}</code>
      </div>
      <el-button class="totp-enrollment-dialog__copy" @click="copyRecoveryCodes">
        <el-icon><RiFileCopyFill /></el-icon>
        复制恢复码
      </el-button>
    </section>

    <template #footer>
      <el-button v-if="step !== 'recovery'" :disabled="submitting" @click="close">取消</el-button>
      <el-button
        v-if="step === 'reauth'"
        type="primary"
        :loading="submitting"
        @click="startEnrollment"
      >
        继续
      </el-button>
      <el-button
        v-else-if="step === 'scan'"
        type="primary"
        :loading="submitting"
        @click="confirmEnrollment"
      >
        确认开启
      </el-button>
      <el-button v-else type="primary" @click="complete">我已保存，完成</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.totp-enrollment-dialog) {
  --el-dialog-padding-primary: 0;

  border-radius: var(--el-border-radius-large);
}

:global(.totp-enrollment-dialog .el-dialog__body) {
  padding: 0;
}

:global(.totp-enrollment-dialog .el-dialog__footer) {
  display: flex;
  justify-content: flex-end;
  gap: var(--el-space-md);
  padding: var(--el-space-lg) var(--el-space-3xl) var(--el-space-2xl);
}

:global(.totp-enrollment-dialog .el-dialog__footer .el-button) {
  min-width: 72px;
  height: 38px;
  margin: 0;
}

.totp-enrollment-dialog {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 56px;
    padding: 0 var(--el-space-3xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-large);
    font-weight: 600;
    line-height: 26px;
  }

  &__header > button {
    display: grid;
    width: 32px;
    height: 32px;
    padding: 0;
    border: 0;
    border-radius: var(--el-border-radius-medium);
    place-items: center;
    background: transparent;
    color: var(--el-text-color-regular);
    cursor: pointer;
    font-size: var(--el-font-size-extra-large);
  }

  &__header > button:hover {
    background: var(--el-fill-color-light);
  }

  &__content {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: var(--el-space-3xl);
    color: var(--el-text-color-regular);
  }

  &__content > p {
    width: 100%;
    margin: 0 0 var(--el-space-xl);
    line-height: 22px;
  }

  &__content > .el-form {
    width: 100%;
  }

  &__hero {
    margin-bottom: var(--el-space-lg);
    color: var(--el-color-primary);
    font-size: 42px;
  }

  &__qr {
    width: 220px;
    height: 220px;
    margin: -4px 0 var(--el-space-xl);
  }

  &__warning {
    color: var(--el-color-warning-dark-2);
  }

  &__codes {
    display: grid;
    width: 100%;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-md);
    padding: var(--el-space-lg);
    border-radius: var(--el-border-radius-medium);
    background: var(--el-fill-color-light);
  }

  &__codes > code {
    color: var(--el-text-color-primary);
    font-family: var(--el-font-family);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__copy {
    align-self: flex-start;
    margin-top: var(--el-space-lg);
  }
}
</style>
