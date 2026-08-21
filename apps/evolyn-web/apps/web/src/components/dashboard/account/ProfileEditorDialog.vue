<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus';
import { reactive, ref, watch } from 'vue';
import type { DeepReadonly } from 'vue';
import type { AccountInfo } from '~/types';
import type { AccountProfileForm } from '~/types/account';

defineOptions({ name: 'ProfileEditorDialog' });

const props = defineProps<{
  modelValue: boolean;
  account?: DeepReadonly<AccountInfo>;
  loading?: boolean;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  submit: [payload: AccountProfileForm];
}>();

const formRef = ref<FormInstance>();
const form = reactive<AccountProfileForm>({
  nickname: '',
  email: '',
  avatar: '',
});

const rules: FormRules<AccountProfileForm> = {
  nickname: [
    { required: true, message: '请输入姓名', trigger: 'blur' },
    { max: 64, message: '姓名不能超过 64 个字符', trigger: 'blur' },
  ],
  email: [{ type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' }],
  avatar: [{ type: 'url', message: '请输入有效的头像地址', trigger: 'blur' }],
};

// 每次打开时都以最新账号聚合信息回填，避免在别处改名后弹窗展示旧值。
watch(
  () => [props.modelValue, props.account] as const,
  ([visible, account]) => {
    if (!visible) return;
    form.nickname = account?.nickname ?? '';
    form.email = account?.email ?? '';
    form.avatar = account?.avatar ?? '';
    formRef.value?.clearValidate();
  },
  { immediate: true },
);

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;
  emit('submit', {
    nickname: form.nickname.trim(),
    // 后端当前采用非空字段更新；空值不出参，避免误导为支持解绑邮箱或清空头像。
    email: form.email?.trim() || undefined,
    avatar: form.avatar?.trim() || undefined,
  });
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="编辑个人资料"
    width="460px"
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="78px" @submit.prevent="submit">
      <el-form-item label="姓名" prop="nickname">
        <el-input v-model="form.nickname" maxlength="64" show-word-limit autocomplete="nickname" />
      </el-form-item>
      <el-form-item label="邮箱" prop="email">
        <el-input v-model="form.email" type="email" autocomplete="email" />
      </el-form-item>
      <el-form-item label="头像地址" prop="avatar">
        <el-input v-model="form.avatar" placeholder="https://example.com/avatar.png" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button :disabled="loading" @click="emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>
