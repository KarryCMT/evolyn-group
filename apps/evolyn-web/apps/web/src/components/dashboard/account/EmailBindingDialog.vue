<script setup lang="ts">
import type { DeepReadonly } from 'vue';
import type { AccountInfo } from '~/types';
import { RiArrowDownSFill, RiCloseFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, onBeforeUnmount, reactive, shallowRef, watch } from 'vue';
import { bindMyEmail, sendMyEmailCode, verifyMyEmailIdentity } from '~/api/account';
import { sendSmsCode } from '~/api/auth';

defineOptions({ name: 'EmailBindingDialog' });

const props = defineProps<{
  modelValue: boolean;
  account?: DeepReadonly<AccountInfo>;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  bound: [];
}>();

type BindingStep = 'verify-phone' | 'bind-email';

const step = shallowRef<BindingStep>('verify-phone');
const sendingPhoneCode = shallowRef(false);
const verifyingIdentity = shallowRef(false);
const sendingEmailCode = shallowRef(false);
const saving = shallowRef(false);
const phoneResendSeconds = shallowRef(0);
const emailResendSeconds = shallowRef(0);
const verificationToken = shallowRef('');
let phoneTimer: number | undefined;
let emailTimer: number | undefined;

const form = reactive({
  phoneCode: '',
  email: '',
  emailCode: '',
});

const phone = computed(() => props.account?.phone || '');
const phoneCodeText = computed(() =>
  phoneResendSeconds.value ? `${phoneResendSeconds.value}s 后重试` : '获取验证码',
);
const emailCodeText = computed(() =>
  emailResendSeconds.value ? `${emailResendSeconds.value}s 后重试` : '获取验证码',
);

function clearTimer(timer: 'phone' | 'email') {
  const timerId = timer === 'phone' ? phoneTimer : emailTimer;
  if (timerId !== undefined) window.clearInterval(timerId);
  if (timer === 'phone') {
    phoneTimer = undefined;
    phoneResendSeconds.value = 0;
  } else {
    emailTimer = undefined;
    emailResendSeconds.value = 0;
  }
}

function startCountdown(timer: 'phone' | 'email') {
  clearTimer(timer);
  if (timer === 'phone') {
    phoneResendSeconds.value = 60;
    phoneTimer = window.setInterval(() => {
      phoneResendSeconds.value -= 1;
      if (phoneResendSeconds.value <= 0) clearTimer('phone');
    }, 1000);
    return;
  }
  emailResendSeconds.value = 60;
  emailTimer = window.setInterval(() => {
    emailResendSeconds.value -= 1;
    if (emailResendSeconds.value <= 0) clearTimer('email');
  }, 1000);
}

function close() {
  emit('update:modelValue', false);
}

async function sendPhoneCode() {
  if (!phone.value) {
    ElMessage.warning('当前账号未绑定手机号，暂无法完成身份验证');
    return;
  }
  if (sendingPhoneCode.value || phoneResendSeconds.value > 0) return;

  sendingPhoneCode.value = true;
  try {
    // rebind/old 仅允许向当前账号已绑定的手机号发送，避免成为短信骚扰入口。
    await sendSmsCode(phone.value, 'rebind', 'old');
    startCountdown('phone');
    ElMessage.success('验证码已发送，请注意查收短信');
  } finally {
    sendingPhoneCode.value = false;
  }
}

async function continueToEmail() {
  if (!/^\d{6}$/.test(form.phoneCode.trim())) {
    ElMessage.warning('请输入 6 位短信验证码');
    return;
  }
  if (verifyingIdentity.value) return;

  verifyingIdentity.value = true;
  try {
    const result = await verifyMyEmailIdentity(form.phoneCode.trim());
    verificationToken.value = result.verificationToken;
    step.value = 'bind-email';
  } finally {
    verifyingIdentity.value = false;
  }
}

async function sendEmailCode() {
  if (!/^\S+@\S+\.\S+$/.test(form.email.trim())) {
    ElMessage.warning('请输入正确的邮箱地址');
    return;
  }
  if (emailResendSeconds.value > 0 || sendingEmailCode.value) return;

  sendingEmailCode.value = true;
  try {
    await sendMyEmailCode({
      email: form.email.trim(),
      verificationToken: verificationToken.value,
    });
    startCountdown('email');
    ElMessage.success('验证码已发送，请注意查收邮箱');
  } finally {
    sendingEmailCode.value = false;
  }
}

async function submit() {
  if (!/^\S+@\S+\.\S+$/.test(form.email.trim())) {
    ElMessage.warning('请输入正确的邮箱地址');
    return;
  }
  if (!/^\d{6}$/.test(form.emailCode.trim())) {
    ElMessage.warning('请输入 6 位邮箱验证码');
    return;
  }
  if (saving.value) return;

  saving.value = true;
  try {
    await bindMyEmail({
      email: form.email.trim(),
      emailCode: form.emailCode.trim(),
      verificationToken: verificationToken.value,
    });
    ElMessage.success('邮箱绑定成功');
    emit('bound');
  } finally {
    saving.value = false;
  }
}

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return;
    step.value = 'verify-phone';
    form.phoneCode = '';
    form.email = props.account?.email || '';
    form.emailCode = '';
    verificationToken.value = '';
  },
);

onBeforeUnmount(() => {
  clearTimer('phone');
  clearTimer('email');
});
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    width="720px"
    class="email-binding-dialog"
    :show-close="false"
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template #header>
      <div class="email-binding-dialog__header">
        <span>绑定邮箱</span>
        <button type="button" aria-label="关闭绑定邮箱" @click="close"><RiCloseFill /></button>
      </div>
    </template>

    <section v-if="step === 'verify-phone'" class="email-binding-dialog__body">
      <p class="email-binding-dialog__description">
        为了你的账户安全，请进行身份验证。验证成功后才可进行下一步操作
      </p>
      <div class="email-binding-dialog__field">
        <label>当前手机号</label>
        <el-input :model-value="phone" disabled>
          <template #prepend>
            <span class="email-binding-dialog__dial-code">+86 <RiArrowDownSFill /></span>
          </template>
        </el-input>
      </div>
      <div class="email-binding-dialog__field">
        <label>验证码</label>
        <el-input
          v-model="form.phoneCode"
          maxlength="6"
          inputmode="numeric"
          placeholder="短信验证码"
        >
          <template #append>
            <el-button
              class="email-binding-dialog__code-button"
              :loading="sendingPhoneCode"
              :disabled="phoneResendSeconds > 0 || !phone"
              @click="sendPhoneCode"
            >
              {{ phoneCodeText }}
            </el-button>
          </template>
        </el-input>
      </div>
    </section>

    <section v-else class="email-binding-dialog__body email-binding-dialog__body--email">
      <div class="email-binding-dialog__field">
        <label>新邮箱</label>
        <el-input v-model="form.email" autocomplete="email" />
      </div>
      <div class="email-binding-dialog__field">
        <label>验证码</label>
        <el-input
          v-model="form.emailCode"
          maxlength="6"
          inputmode="numeric"
          placeholder="邮件验证码"
        >
          <template #append>
            <el-button
              class="email-binding-dialog__code-button"
              :loading="sendingEmailCode"
              :disabled="emailResendSeconds > 0 || !verificationToken"
              @click="sendEmailCode"
            >
              {{ emailCodeText }}
            </el-button>
          </template>
        </el-input>
      </div>
    </section>

    <template #footer>
      <el-button :disabled="saving" @click="close">取消</el-button>
      <el-button
        v-if="step === 'verify-phone'"
        type="primary"
        :loading="verifyingIdentity"
        @click="continueToEmail"
      >
        下一步
      </el-button>
      <el-button v-else type="primary" :loading="saving" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.email-binding-dialog) {
  --el-dialog-padding-primary: 0;

  border-radius: var(--el-border-radius-base);
}

.email-binding-dialog {
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
    border-radius: var(--el-border-radius-small);
    background: transparent;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    font-size: 22px;
  }

  &__header button:hover {
    background: var(--el-fill-color-light);
  }

  &__body {
    min-height: 316px;
    padding: 28px 24px 0;
  }

  &__body--email {
    padding-top: 34px;
  }

  &__description {
    margin: 0 0 28px;
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__field {
    margin-bottom: 22px;
  }

  &__field label {
    display: block;
    margin-bottom: 8px;
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__field :deep(.el-input__wrapper) {
    min-height: 40px;
  }

  &__field :deep(.el-input-group__prepend) {
    padding: 0 12px;
    color: var(--el-text-color-secondary);
  }

  &__field :deep(.el-input-group__append) {
    padding: 0;
  }

  &__dial-code {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  &__code-button {
    min-width: 146px;
    height: 38px;
    border: 0;
    border-radius: 0;
    color: var(--el-text-color-primary);
  }
}

:global(.email-binding-dialog .el-dialog__body) {
  padding: 0;
}

:global(.email-binding-dialog .el-dialog__footer) {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 12px 24px 22px;
}

:global(.email-binding-dialog .el-dialog__footer .el-button) {
  min-width: 72px;
  height: 38px;
  margin: 0;
}

@media (max-width: 768px) {
  :global(.email-binding-dialog) {
    width: calc(100% - 32px) !important;
    margin: 8vh auto 0;
  }

  .email-binding-dialog {
    &__header {
      height: 52px;
      padding: 0 20px;
      font-size: 18px;
      line-height: 26px;
    }

    &__body {
      min-height: 0;
      padding: 24px 20px 0;
    }
  }

  :global(.email-binding-dialog .el-dialog__footer) {
    padding-right: 20px;
    padding-left: 20px;
  }
}
</style>
