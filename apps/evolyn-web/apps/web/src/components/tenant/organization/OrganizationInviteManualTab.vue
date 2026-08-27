<script setup lang="ts">
import { RiAddFill } from '@remixicon/vue';
import { ElTreeSelect } from 'element-plus';
import type { MemberInvitationForm } from '~/composables/tenant/useMemberInvitation';

interface DepartmentOption {
  value: number;
  label: string;
  children?: DepartmentOption[];
}

defineProps<{
  departments: DepartmentOption[];
  submitting: boolean;
}>();

const form = defineModel<MemberInvitationForm>('form', { required: true });

const emit = defineEmits<{
  submit: [];
  clear: [];
}>();
</script>

<template>
  <section class="organization-invite-manual" aria-label="手动添加成员">
    <ul class="organization-invite-manual__tips">
      <li>成员加入时无法更改你输入的内容，请输入正确的信息</li>
      <li>按输入的联系方式发送邀请；同时输入手机和邮箱时，只发送手机邀请</li>
    </ul>

    <el-form
      class="organization-invite-manual__form"
      label-position="top"
      @submit.prevent="emit('submit')"
    >
      <div class="organization-invite-manual__grid">
        <el-form-item label="姓名" required>
          <el-input v-model="form.name" maxlength="80" placeholder="必填，最长80个字符" />
        </el-form-item>
        <el-form-item label="编号">
          <el-input
            v-model="form.identifier"
            maxlength="50"
            placeholder="成员在企业内唯一标识，如工号"
          />
        </el-form-item>
        <el-form-item label="手机">
          <el-input v-model="form.phone" maxlength="32" placeholder="手机和邮箱必填一项">
            <template #prepend>+86</template>
          </el-input>
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="form.email" maxlength="256" placeholder="手机和邮箱必填一项" />
        </el-form-item>
      </div>

      <el-form-item label="部门" class="organization-invite-manual__department">
        <el-tree-select
          v-model="form.departmentIds"
          class="organization-invite-manual__department-select"
          :data="departments"
          multiple
          check-strictly
          show-checkbox
          node-key="value"
          :render-after-expand="false"
          placeholder="选择部门"
          :props="{ label: 'label', children: 'children' }"
        >
          <template #default>
            <span><RiAddFill />选择部门</span>
          </template>
        </el-tree-select>
      </el-form-item>

      <div class="organization-invite-manual__grid organization-invite-manual__grid--extended">
        <el-form-item label="别名"
          ><el-input v-model="form.alias" maxlength="50" placeholder="选填"
        /></el-form-item>
        <el-form-item label="工号"
          ><el-input v-model="form.employeeNo" maxlength="50" placeholder="选填"
        /></el-form-item>
        <el-form-item label="性别"
          ><el-input v-model="form.gender" maxlength="50" placeholder="选填"
        /></el-form-item>
        <el-form-item label="职务"
          ><el-input v-model="form.title" maxlength="50" placeholder="选填"
        /></el-form-item>
        <el-form-item label="聘用形式"
          ><el-input v-model="form.employmentType" maxlength="50" placeholder="选填"
        /></el-form-item>
        <el-form-item label="入职日期"
          ><el-date-picker
            v-model="form.hiredAt"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="选择日期"
        /></el-form-item>
        <el-form-item label="工作地点"
          ><el-input v-model="form.workLocation" maxlength="50" placeholder="选填"
        /></el-form-item>
        <el-form-item label="出生日期"
          ><el-date-picker
            v-model="form.birthday"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="选择日期"
        /></el-form-item>
        <el-form-item label="学历"
          ><el-input v-model="form.education" maxlength="50" placeholder="选填"
        /></el-form-item>
      </div>
    </el-form>

    <footer class="organization-invite-manual__footer">
      <el-button size="large" @click="emit('clear')">清空</el-button>
      <el-button type="primary" size="large" :loading="submitting" @click="emit('submit')"
        >邀请</el-button
      >
    </footer>
  </section>
</template>

<style scoped lang="scss">
.organization-invite-manual {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;

  &__tips {
    margin: 0 0 var(--el-space-3xl);
    padding: var(--el-space-lg) var(--el-space-2xl)
      var(--el-space-lg) var(--el-space-5xl);
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__form {
    flex: 1;
  }

  &__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: var(--el-space-3xl);
  }

  &__grid--extended {
    margin-top: var(--el-space-md);
  }

  &__department {
    margin-top: var(--el-space-sm);
  }

  &__department-select {
    width: 100%;
  }

  &__department-select :deep(.el-select__wrapper) {
    min-height: 88px;
    border: 1px dashed var(--el-border-color);
    box-shadow: none;
  }

  &__department-select :deep(.el-select__placeholder) {
    margin: auto;
  }

  &__department-select :deep(.el-select__placeholder span) {
    display: inline-flex;
    align-items: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-medium);
  }

  &__department-select :deep(.el-select__placeholder svg) {
    width: 22px;
    height: 22px;
  }

  &__footer {
    display: flex;
    padding: var(--el-space-3xl) 0 var(--el-space-xs);
    border-top: 1px solid var(--el-border-color-lighter);
    justify-content: flex-end;
    gap: var(--el-space-2xl);
  }

  &__footer .el-button {
    min-width: 108px;
  }
}

@media (max-width: 820px) {
  .organization-invite-manual__grid {
    grid-template-columns: 1fr;
  }
}
</style>
