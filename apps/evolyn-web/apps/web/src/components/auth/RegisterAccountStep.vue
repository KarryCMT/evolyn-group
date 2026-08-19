<script setup lang="ts">
// 注册向导第 1 步：账号信息。只负责收集与校验，通过「下一步」上抛；
// 须勾选服务条款与隐私政策后方可继续
import { reactive, useTemplateRef } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { Iphone, Lock, Message, User } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

defineProps<{
  /** 提交中：按钮显示 loading 并防重复提交 */
  loading?: boolean
}>()

const emit = defineEmits<{
  /** 校验通过后上抛账号信息，进入下一步 */
  next: [payload: { name: string; phone: string; email?: string; password: string }]
}>()

const formRef = useTemplateRef<FormInstance>('formRef')

const form = reactive({
  name: '',
  phone: '',
  email: '',
  password: '',
  confirmPassword: '',
  agreed: false,
})

const rules: FormRules = {
  name: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 30, message: '长度需为 3 - 30 个字符', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_-]+$/, message: '仅支持字母、数字、下划线和连字符', trigger: 'blur' },
  ],
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur' },
  ],
  email: [{ type: 'email', message: '邮箱格式不正确', trigger: 'blur' }],
  password: [
    { required: true, message: '请设置密码', trigger: 'blur' },
    { min: 6, max: 20, message: '长度需为 6 - 20 个字符', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      // 与首次输入的密码一致性校验
      validator: (_rule, value: string, callback) => {
        if (value !== form.password) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
}

async function handleNext() {
  const valid = await formRef.value?.validate().then(() => true, () => false)
  if (!valid) return

  if (!form.agreed) {
    ElMessage.warning('请先阅读并同意服务条款与隐私政策')
    return
  }

  emit('next', {
    name: form.name.trim(),
    phone: form.phone.trim(),
    email: form.email.trim() || undefined,
    password: form.password,
  })
}
</script>

<template>
  <el-form
    ref="formRef"
    class="account-step"
    :model="form"
    :rules="rules"
    size="large"
    label-position="top"
    @submit.prevent="handleNext"
  >
    <el-form-item prop="name">
      <el-input
        v-model="form.name"
        name="name"
        placeholder="请输入用户名"
        autocomplete="username"
        clearable
        :prefix-icon="User"
      />
    </el-form-item>

    <el-form-item prop="phone">
      <el-input
        v-model="form.phone"
        name="phone"
        placeholder="请输入手机号"
        autocomplete="tel"
        clearable
        :prefix-icon="Iphone"
      >
        <template #prepend>+86</template>
      </el-input>
    </el-form-item>

    <el-form-item prop="email">
      <el-input
        v-model="form.email"
        name="email"
        placeholder="请输入邮箱（选填）"
        autocomplete="email"
        clearable
        :prefix-icon="Message"
      />
    </el-form-item>

    <el-form-item prop="password">
      <el-input
        v-model="form.password"
        name="new-password"
        type="password"
        placeholder="设置密码（6 - 20 位）"
        autocomplete="new-password"
        show-password
        :prefix-icon="Lock"
      />
    </el-form-item>

    <el-form-item prop="confirmPassword">
      <el-input
        v-model="form.confirmPassword"
        name="confirm-password"
        type="password"
        placeholder="请再次输入密码"
        autocomplete="new-password"
        show-password
        :prefix-icon="Lock"
        @keyup.enter="handleNext"
      />
    </el-form-item>

    <el-checkbox v-model="form.agreed" class="account-step__agreement">
      我已阅读并同意
      <a class="account-step__link" @click.prevent="ElMessage.info('服务条款文档即将上线')">《服务条款》</a>
      与
      <a class="account-step__link" @click.prevent="ElMessage.info('隐私政策文档即将上线')">《隐私政策》</a>
    </el-checkbox>

    <el-button
      class="account-step__next"
      type="primary"
      size="large"
      native-type="submit"
      :loading="loading"
    >
      下一步
    </el-button>
  </el-form>
</template>

<style lang="scss" scoped>
.account-step__agreement {
  margin: 4px 0 20px;
  height: auto;
  white-space: normal;
  line-height: 1.6;
  align-items: flex-start;
}

.account-step__link {
  color: var(--el-color-primary);

  &:hover {
    color: var(--el-color-primary-light-3);
  }
}

.account-step__next {
  width: 100%;
}
</style>
