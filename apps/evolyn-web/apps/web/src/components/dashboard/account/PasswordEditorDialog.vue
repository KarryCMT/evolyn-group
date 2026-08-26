<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus';
import { RiCloseFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, onBeforeUnmount, reactive, ref, shallowRef, watch } from 'vue';
import { sendSmsCode } from '~/api/auth';
import type { AccountPasswordForm } from '~/types/account';

defineOptions({ name: 'PasswordEditorDialog' });

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    passwordInitialized: boolean;
    phone: string;
    loading?: boolean;
  }>(),
  { loading: false },
);

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  submit: [payload: AccountPasswordForm];
}>();

type SetupStep = 'verify' | 'password';

const formRef = ref<FormInstance>();
const setupStep = shallowRef<SetupStep>('verify');
const sendingCode = shallowRef(false);
const resendSeconds = shallowRef(0);
let resendTimer: number | undefined;

const form = reactive({
  oldPassword: '',
  smsCode: '',
  newPassword: '',
  confirmPassword: '',
});

const isVerificationStep = computed(
  () => !props.passwordInitialized && setupStep.value === 'verify',
);
const title = computed(() => (props.passwordInitialized ? '修改密码' : '设置密码'));
const sendCodeText = computed(() =>
  resendSeconds.value ? `${resendSeconds.value}s 后重试` : '获取验证码',
);
const rules = computed<FormRules<typeof form>>(() => ({
  oldPassword: props.passwordInitialized
    ? [{ required: true, message: '请输入当前密码', trigger: 'blur' }]
    : [],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, max: 64, message: '密码长度为 8-64 位', trigger: 'blur' },
    {
      // 与后端口径一致：弱口令仍由服务端黑名单统一拦截。
      pattern: /^(?=.*[A-Za-z])(?=.*\d).{8,64}$/,
      message: '密码需同时包含字母和数字',
      trigger: 'blur',
    },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (_rule, value, callback) => {
        callback(value === form.newPassword ? undefined : new Error('两次输入的密码不一致'));
      },
      trigger: 'blur',
    },
  ],
}));

function clearResendTimer() {
  if (resendTimer !== undefined) window.clearInterval(resendTimer);
  resendTimer = undefined;
  resendSeconds.value = 0;
}

function startResendCountdown() {
  clearResendTimer();
  resendSeconds.value = 60;
  resendTimer = window.setInterval(() => {
    resendSeconds.value -= 1;
    if (resendSeconds.value <= 0) clearResendTimer();
  }, 1000);
}

function close() {
  emit('update:modelValue', false);
}

async function sendVerificationCode() {
  if (!props.phone) {
    ElMessage.warning('当前账号未绑定手机号，暂无法设置密码');
    return;
  }
  if (sendingCode.value || resendSeconds.value > 0) return;

  sendingCode.value = true;
  try {
    await sendSmsCode(props.phone, 'reset');
    startResendCountdown();
    ElMessage.success('验证码已发送，请注意查收短信');
  } catch (error) {
    // 请求层会将 429（含 AUTH_SMS_IP_LIMIT）归一为 ApiError，并保留后端的安全提示文案。
    ElMessage.error(error instanceof Error ? error.message : '验证码发送失败，请稍后重试');
  } finally {
    sendingCode.value = false;
  }
}

function continueToPasswordForm() {
  if (!/^\d{6}$/.test(form.smsCode.trim())) {
    ElMessage.warning('请输入 6 位短信验证码');
    return;
  }
  // 验证码会在最终保存时由找回密码接口原子校验，避免验证后被复用或过期。
  setupStep.value = 'password';
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;
  emit('submit', {
    oldPassword: props.passwordInitialized ? form.oldPassword : undefined,
    newPassword: form.newPassword,
    smsCode: props.passwordInitialized ? undefined : form.smsCode.trim(),
  });
}

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return;
    setupStep.value = 'verify';
    form.oldPassword = '';
    form.smsCode = '';
    form.newPassword = '';
    form.confirmPassword = '';
    formRef.value?.clearValidate();
  },
);

onBeforeUnmount(clearResendTimer);
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    width="440px"
    class="password-editor-dialog"
    :show-close="false"
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template #header>
      <div class="password-editor-dialog__header">
        <span>{{ title }}</span>
        <button type="button" aria-label="关闭设置密码" @click="close"><RiCloseFill /></button>
      </div>
    </template>

    <template v-if="isVerificationStep">
      <p class="password-editor-dialog__description">
        为了你的账户安全，请进行身份验证。验证成功后才可进行下一步操作
      </p>
      <el-form class="password-editor-dialog__form" label-position="top">
        <el-form-item label="当前手机号">
          <el-input :model-value="phone" disabled>
            <template #prepend>+86</template>
          </el-input>
        </el-form-item>
        <el-form-item label="验证码">
          <el-input
            v-model="form.smsCode"
            maxlength="6"
            inputmode="numeric"
            placeholder="短信验证码"
          >
            <template #append>
              <el-button
                class="password-editor-dialog__send-code"
                :loading="sendingCode"
                :disabled="resendSeconds > 0 || !phone"
                @click="sendVerificationCode"
              >
                {{ sendCodeText }}
              </el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
    </template>

    <el-form
      v-else
      ref="formRef"
      class="password-editor-dialog__form"
      :model="form"
      :rules="rules"
      label-position="top"
      @submit.prevent="submit"
    >
      <el-form-item v-if="passwordInitialized" label="当前密码" prop="oldPassword">
        <el-input
          v-model="form.oldPassword"
          type="password"
          show-password
          autocomplete="current-password"
          placeholder="当前密码"
        />
      </el-form-item>
      <el-form-item label="新密码" prop="newPassword">
        <el-input
          v-model="form.newPassword"
          type="password"
          show-password
          autocomplete="new-password"
          placeholder="新密码"
        />
      </el-form-item>
      <el-form-item label="确认密码" prop="confirmPassword">
        <el-input
          v-model="form.confirmPassword"
          type="password"
          show-password
          autocomplete="new-password"
          placeholder="确认密码"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button :disabled="loading" @click="close">取消</el-button>
      <el-button v-if="isVerificationStep" type="primary" @click="continueToPasswordForm">
        下一步
      </el-button>
      <el-button v-else type="primary" :loading="loading" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.password-editor-dialog) {
  --el-dialog-padding-primary: 0;

  border-radius: 12px;
}

.password-editor-dialog {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 56px;
    padding: 0 24px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 600;
    line-height: 28px;
  }

  &__header button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    font-size: 22px;
  }

  &__header button:hover {
    background: var(--el-fill-color-light);
  }

  &__description {
    margin: 0;
    padding: 24px 24px 22px;
    color: var(--el-text-color-regular);
    font-size: 14px;
    line-height: 22px;
  }

  &__description + &__form {
    padding-top: 0;
  }

  &__form {
    padding: 24px 24px 0;
  }

  &__form :deep(.el-form-item) {
    margin-bottom: 20px;
  }

  &__form :deep(.el-form-item__label) {
    padding-bottom: 8px;
    color: var(--el-text-color-primary);
    font-size: 14px;
    line-height: 20px;
  }

  &__form :deep(.el-input__wrapper) {
    min-height: 40px;
  }

  &__form :deep(.el-input-group__prepend) {
    min-width: 54px;
    justify-content: center;
  }

  &__send-code {
    min-width: 104px;
  }
}

:global(.password-editor-dialog .el-dialog__body) {
  padding: 0;
}

:global(.password-editor-dialog .el-dialog__footer) {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 12px 24px 22px;
}

:global(.password-editor-dialog .el-dialog__footer .el-button) {
  min-width: 72px;
  height: 38px;
  margin: 0;
}
</style>
