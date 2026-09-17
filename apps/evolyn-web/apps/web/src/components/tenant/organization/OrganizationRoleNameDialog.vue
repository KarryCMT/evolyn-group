<script setup lang="ts">
import { ElDialog, ElInput } from 'element-plus';

defineProps<{
  mode: 'group' | 'role' | 'rename' | 'group-rename';
}>();

const emit = defineEmits<{
  closed: [];
  confirm: [];
}>();
const open = defineModel<boolean>({ required: true });
const name = defineModel<string>('name', { required: true });
</script>

<template>
  <ElDialog
    v-model="open"
    :title="
      mode === 'group'
        ? '创建角色组'
        : mode === 'rename' || mode === 'group-rename'
          ? '修改名称'
          : '创建角色'
    "
    width="480px"
    align-center
    @closed="emit('closed')"
  >
    <ElInput
      v-model="name"
      :placeholder="
        mode === 'group' || mode === 'group-rename' ? '请输入角色组名称' : '请输入角色名称'
      "
      maxlength="30"
      show-word-limit
      @keyup.enter="emit('confirm')"
    />
    <template #footer>
      <el-button @click="open = false">取消</el-button>
      <el-button type="primary" @click="emit('confirm')">确定</el-button>
    </template>
  </ElDialog>
</template>
