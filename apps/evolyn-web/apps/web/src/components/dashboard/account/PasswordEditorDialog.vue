<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus';
import { computed, reactive, ref, watch } from 'vue';
import type { AccountPasswordForm } from '~/types/account';

defineOptions({ name: 'PasswordEditorDialog' });

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    passwordInitialized: boolean;
    loading?: boolean;
  }>(),
  { loading: false },
);

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  submit: [payload: AccountPasswordForm];
}>();

const formRef = ref<FormInstance>();
const form = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
});

const title = computed(() => (props.passwordInitialized ? '修改密码' : '设置密码'));
const rules = computed<FormRules<typeof form>>(() => ({
  oldPassword: props.passwordInitialized
    ? [{ required: true, message: '请输入当前密码', trigger: 'blur' }]
    : [],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' },
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

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return;
    form.oldPassword = '';
    form.newPassword = '';
    form.confirmPassword = '';
    formRef.value?.clearValidate();
  },
);

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;
  emit('submit', {
    oldPassword: props.passwordInitialized ? form.oldPassword : undefined,
    newPassword: form.newPassword,
  });
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    :title="title"
    width="460px"
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-alert
      v-if="!passwordInitialized"
      title="这是首次设置密码，无需填写当前密码。"
      type="info"
      :closable="false"
      show-icon
      class="password-editor-dialog__notice"
    />
    <el-form ref="formRef" :model="form" :rules="rules" label-width="92px" @submit.prevent="submit">
      <el-form-item v-if="passwordInitialized" label="当前密码" prop="oldPassword">
        <el-input
          v-model="form.oldPassword"
          type="password"
          show-password
          autocomplete="current-password"
        />
      </el-form-item>
      <el-form-item label="新密码" prop="newPassword">
        <el-input
          v-model="form.newPassword"
          type="password"
          show-password
          autocomplete="new-password"
        />
      </el-form-item>
      <el-form-item label="确认新密码" prop="confirmPassword">
        <el-input
          v-model="form.confirmPassword"
          type="password"
          show-password
          autocomplete="new-password"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button :disabled="loading" @click="emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.password-editor-dialog__notice {
  margin-bottom: 18px;
}
</style>
