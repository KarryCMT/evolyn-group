<script setup lang="ts">
// 注册向导第 1 步「注册账号」：只收集手机号 + 短信验证码（设计稿口径），
// 不收集用户名/密码。「获取验证码」通过手机号校验后上抛父级发送并启动
// 60s 重发倒计时；提交校验通过后上抛，父级仅暂存推进——注册动作合并到
// 向导第 3 步「进入产品」的最终提交，验证码也在彼时一次性校验
import { onUnmounted, reactive, shallowRef, useTemplateRef } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { Iphone } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

/** 重发倒计时秒数：与后端发送冷却窗口一致 */
const RESEND_SECONDS = 60

const props = defineProps<{
  /** 提交中：按钮显示 loading 并防重复提交 */
  loading?: boolean
  /** 回退场景带回的手机号（验证码过期退回本步时免重填） */
  defaultPhone?: string
}>()

const emit = defineEmits<{
  /** 校验通过后上抛手机号 + 验证码，由父级暂存并推进（注册合并进最终提交） */
  submit: [payload: { phone: string; smsCode: string }]
  /** 请求发送注册短信验证码（先通过手机号校验） */
  'send-code': [phone: string]
}>()

const formRef = useTemplateRef<FormInstance>('formRef')

const form = reactive({
  phone: props.defaultPhone ?? '',
  smsCode: '',
})

const rules: FormRules = {
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur' },
  ],
  smsCode: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    { pattern: /^\d{6}$/, message: '验证码为 6 位数字', trigger: 'blur' },
  ],
}

// 重发倒计时：发送即启动，到 0 自动恢复按钮
const countdown = shallowRef(0)
let timer: ReturnType<typeof setInterval> | undefined

function startCountdown() {
  countdown.value = RESEND_SECONDS
  timer = setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0) {
      clearInterval(timer)
      timer = undefined
    }
  }, 1000)
}

onUnmounted(() => {
  if (timer !== undefined) clearInterval(timer)
})

async function handleSubmit() {
  const valid = await formRef.value?.validate().then(() => true, () => false)
  if (!valid) return

  emit('submit', { phone: form.phone.trim(), smsCode: form.smsCode })
}

function handleSendCode() {
  // 仅校验手机号字段，通过后上抛发送请求并进入倒计时（频率由后端冷却兜底）
  formRef.value
    ?.validateField('phone')
    .then(() => {
      emit('send-code', form.phone.trim())
      startCountdown()
    })
    .catch(() => {})
}
</script>

<template>
  <el-form
    ref="formRef"
    class="account-step"
    :model="form"
    :rules="rules"
    size="large"
    @submit.prevent="handleSubmit"
  >
    <el-form-item prop="phone">
      <el-input
        v-model="form.phone"
        name="phone"
        placeholder="你的手机号"
        autocomplete="tel"
        clearable
        :prefix-icon="Iphone"
      >
        <template #prepend>+86</template>
      </el-input>
    </el-form-item>

    <el-form-item prop="smsCode">
      <el-input
        v-model="form.smsCode"
        name="sms-code"
        placeholder="收到的验证码"
        autocomplete="one-time-code"
        maxlength="6"
        @keyup.enter="handleSubmit"
      >
        <!-- 获取验证码：输入框内右侧文字按钮（设计稿口径），倒计时中禁用 -->
        <template #suffix>
          <button
            class="account-step__send"
            type="button"
            :disabled="countdown > 0 || loading"
            @click="handleSendCode"
          >
            {{ countdown > 0 ? `${countdown}s 后重发` : '获取验证码' }}
          </button>
        </template>
      </el-input>
    </el-form-item>

    <el-button
      class="account-step__register"
      type="primary"
      size="large"
      native-type="submit"
      :loading="loading"
    >
      注册
    </el-button>

    <!-- 协议口径（设计稿）：点击注册即视为同意，不做勾选框拦截 -->
    <p class="account-step__agreement">
      点击注册表明你已阅读并同意
      <a class="account-step__link" @click.prevent="ElMessage.info('服务条款文档即将上线')">《服务条款》</a>
      和
      <a class="account-step__link" @click.prevent="ElMessage.info('隐私声明文档即将上线')">《隐私声明》</a>
    </p>

    <div class="account-step__help">
      <a class="account-step__link" @click.prevent="ElMessage.info('如遇手机号无法注册，请联系客服处理')">手机号无法注册？点击此处</a>
    </div>
  </el-form>
</template>

<style lang="scss" scoped>
// 获取验证码：输入框内右侧无边框文字按钮，与后端冷却窗口联动禁用
.account-step__send {
  padding: 0;
  font-size: var(--el-font-size-base);
  color: var(--el-color-primary);
  background: none;
  border: none;
  cursor: pointer;

  &:hover:not(:disabled) {
    color: var(--el-color-primary-light-3);
  }

  &:disabled {
    color: var(--el-text-color-placeholder);
    cursor: not-allowed;
  }
}

.account-step__register {
  width: 100%;
}

// 协议说明：主按钮下方居中的辅助文案（设计稿口径）
.account-step__agreement {
  margin: 12px 0 0;
  font-size: var(--el-font-size-small);
  line-height: 1.6;
  text-align: center;
  color: var(--el-text-color-secondary);
}

.account-step__help {
  margin-top: 16px;
  text-align: center;
}

.account-step__link {
  color: var(--el-color-primary);
  cursor: pointer;

  &:hover {
    color: var(--el-color-primary-light-3);
  }
}
</style>
