<script setup lang="ts">
// 密码登录表单：手机号 + 密码，只负责收集与校验，提交结果由父级（登录页）处理；
// 「下次自动登录」勾选时本地记住手机号并持久化令牌，取消勾选则为会话级登录
import { reactive, useTemplateRef } from 'vue';
import type { FormInstance, FormRules } from 'element-plus';
import {
  RiLockPasswordFill,
  RiLockPasswordLine,
  RiSmartphoneFill,
  RiSmartphoneLine,
} from '@remixicon/vue';

/** 本地记住手机号的存储键 */
const REMEMBER_PHONE_KEY = 'evolyn.login.phone';

const props = defineProps<{
  /** 提交中：按钮显示 loading 并防重复提交 */
  loading?: boolean;
}>();

const emit = defineEmits<{
  /** 校验通过后上抛登录凭据 */
  submit: [payload: { phone: string; password: string; remember: boolean }];
}>();

const formRef = useTemplateRef<FormInstance>('formRef');

// 表单字段逐项变更，用 reactive 维护
const form = reactive({
  phone: localStorage.getItem(REMEMBER_PHONE_KEY) ?? '',
  password: '',
  remember: localStorage.getItem(REMEMBER_PHONE_KEY) !== null,
});

// 校验规则：与后端账号级校验对齐（手机号 + 密码必填）
const rules: FormRules = {
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur' },
  ],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
};

async function handleSubmit() {
  if (props.loading) return;
  const valid = await formRef.value?.validate().then(
    () => true,
    () => false,
  );
  if (!valid) return;

  // 勾选「下次自动登录」记住手机号，否则清除历史记录
  if (form.remember) {
    localStorage.setItem(REMEMBER_PHONE_KEY, form.phone);
  } else {
    localStorage.removeItem(REMEMBER_PHONE_KEY);
  }

  emit('submit', { phone: form.phone.trim(), password: form.password, remember: form.remember });
}
</script>

<template>
  <el-form
    ref="formRef"
    class="password-login-form"
    :model="form"
    :rules="rules"
    size="large"
    @submit.prevent="handleSubmit"
  >
    <el-form-item prop="phone">
      <el-input
        v-model="form.phone"
        name="phone"
        placeholder="请输入手机号"
        autocomplete="tel"
        clearable
        :prefix-icon="RiSmartphoneLine"
      >
        <template #prepend><span class="auth-phone-prefix">+86</span></template>
      </el-input>
    </el-form-item>

    <el-form-item prop="password">
      <el-input
        v-model="form.password"
        name="password"
        type="password"
        placeholder="请输入密码"
        autocomplete="current-password"
        show-password
        :prefix-icon="RiLockPasswordLine"
        @keyup.enter="handleSubmit"
      />
    </el-form-item>

    <div class="password-login-form__extra">
      <el-checkbox v-model="form.remember">下次自动登录</el-checkbox>
      <router-link class="password-login-form__forgot" to="/auth/forgot-password"
        >忘记密码？</router-link
      >
    </div>

    <el-button
      class="password-login-form__submit"
      type="primary"
      size="large"
      native-type="submit"
      :loading="loading"
      :disabled="loading"
    >
      登录
    </el-button>

    <!-- 登录方式切换（如「验证码登录」）由父级通过 footer 插槽注入 -->
    <slot name="footer" />
  </el-form>
</template>

<style lang="scss" scoped>
.password-login-form__extra {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.password-login-form__forgot {
  font-size: var(--el-font-size-base);
  color: var(--el-color-primary);

  &:hover {
    color: var(--el-color-primary-light-3);
  }
}

.password-login-form__submit {
  width: 100%;
}

.auth-phone-prefix {
  color: var(--el-color-primary);
}
</style>
