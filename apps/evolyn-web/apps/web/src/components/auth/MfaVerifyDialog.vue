<script setup lang="ts">
import { computed, reactive, shallowRef, watch } from 'vue';

const props = defineProps<{
  modelValue: boolean;
  loading?: boolean;
}>();

const emit = defineEmits<{
  'update:modelValue': [visible: boolean];
  submit: [payload: { method: 'totp' | 'recovery'; code: string }];
}>();

const form = reactive({ method: 'totp' as 'totp' | 'recovery', code: '' });
const dialogVisible = computed({
  get: () => props.modelValue,
  set: (visible: boolean) => emit('update:modelValue', visible),
});
const codeLabel = computed(() => (form.method === 'totp' ? '验证器动态码' : '恢复码'));
const codePlaceholder = computed(() =>
  form.method === 'totp' ? '请输入 6 位动态码' : '请输入恢复码',
);
const formRef = shallowRef<{ validate: () => Promise<boolean> }>();

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      form.method = 'totp';
      form.code = '';
    }
  },
);

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;
  emit('submit', { method: form.method, code: form.code.trim() });
}
</script>

<template>
  <el-dialog
    v-model="dialogVisible"
    title="登录二次验证"
    width="400px"
    :close-on-click-modal="false"
  >
    <p class="mfa-verify-dialog__hint">为保护账号安全，请完成验证后继续登录。</p>
    <el-form ref="formRef" :model="form" label-position="top" @submit.prevent="handleSubmit">
      <el-form-item label="验证方式">
        <el-radio-group v-model="form.method">
          <el-radio-button value="totp">验证器</el-radio-button>
          <el-radio-button value="recovery">恢复码</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item
        :label="codeLabel"
        prop="code"
        :rules="[{ required: true, message: `请填写${codeLabel}`, trigger: 'blur' }]"
      >
        <el-input v-model="form.code" :placeholder="codePlaceholder" autocomplete="one-time-code" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button :disabled="loading" @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="loading" @click="handleSubmit">验证并登录</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.mfa-verify-dialog {
  &__hint {
    margin: 0 0 var(--el-space-xl);
    color: var(--el-text-color-secondary);
    line-height: 22px;
  }
}
</style>
