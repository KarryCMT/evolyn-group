<script setup lang="ts">
import { reactive, watch } from 'vue';

export interface TenantProfileCompletion {
  companyName: string;
  companySize: string;
  industry: string;
  managementNeeds: string[];
  role: string;
}

const props = defineProps<{
  initialValue: TenantProfileCompletion;
}>();

const visible = defineModel<boolean>('visible', { required: true });

const emit = defineEmits<{
  save: [profile: TenantProfileCompletion];
}>();

const form = reactive<TenantProfileCompletion>({ ...props.initialValue });

const industryOptions = ['互联网/软件', '制造业', '零售/电商', '教育培训', '专业服务', '其他'];
const sizeOptions = ['1-10', '11-50', '51-200', '201-500', '500+'];
const roleOptions = ['CEO/老板', '业务总监/经理/主管', 'IT/信息化人员', '普通成员', '老师', '学生'];
const needOptions = ['IT项目', '任务', 'CRM/销售', 'OA', '进销存', '人事', '合同', '售后'];

watch(
  () => props.initialValue,
  (value) => Object.assign(form, value),
  { deep: true },
);

function close() {
  visible.value = false;
}

function save() {
  if (!form.companyName.trim()) return;
  emit('save', {
    ...form,
    companyName: form.companyName.trim(),
    managementNeeds: [...form.managementNeeds],
  });
  close();
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="tenant-profile-completion-dialog"
    title="完善信息"
    width="min(960px, calc(100vw - 32px))"
    align-center
    destroy-on-close
    @closed="Object.assign(form, props.initialValue)"
  >
    <el-scrollbar class="tenant-profile-completion-dialog__scrollbar">
      <el-form class="tenant-profile-completion-dialog__form" label-position="top">
        <el-form-item label="你的职位">
          <el-radio-group v-model="form.role" class="tenant-profile-completion-dialog__role-list">
            <el-radio v-for="option in roleOptions" :key="option" :value="option">{{
              option
            }}</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="公司名称" required>
          <el-input v-model="form.companyName" maxlength="50" show-word-limit />
        </el-form-item>

        <el-form-item label="所属行业">
          <el-select v-model="form.industry" class="tenant-profile-completion-dialog__full-control">
            <el-option
              v-for="option in industryOptions"
              :key="option"
              :label="option"
              :value="option"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="公司人数">
          <el-radio-group
            v-model="form.companySize"
            class="tenant-profile-completion-dialog__size-list"
          >
            <el-radio-button v-for="option in sizeOptions" :key="option" :value="option">
              {{ option }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="公司管理需求">
          <el-checkbox-group
            v-model="form.managementNeeds"
            class="tenant-profile-completion-dialog__needs-list"
          >
            <el-checkbox v-for="option in needOptions" :key="option" :value="option">{{
              option
            }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
    </el-scrollbar>

    <template #footer>
      <div class="tenant-profile-completion-dialog__footer">
        <el-button @click="close">取消</el-button>
        <el-button type="primary" :disabled="!form.companyName.trim()" @click="save"
          >确定</el-button
        >
      </div>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.tenant-profile-completion-dialog {
  &__scrollbar {
    max-height: min(60vh, 560px);
    padding-right: 18px;
  }

  &__form {
    padding: 2px 4px 20px;

    :deep(.el-form-item) {
      margin-bottom: 28px;
    }

    :deep(.el-form-item__label) {
      padding-bottom: 10px;
      color: var(--el-text-color-primary);
      font-size: 16px;
      line-height: 24px;
    }

    :deep(.el-input__wrapper),
    :deep(.el-select__wrapper) {
      min-height: 42px;
    }
  }

  &__full-control {
    width: 100%;
  }

  &__role-list,
  &__needs-list {
    display: flex;
    gap: 14px 28px;
  }

  &__role-list {
    flex-wrap: wrap;

    :deep(.el-radio) {
      margin-right: 0;
    }
  }

  &__size-list {
    display: grid;
    width: 100%;
    grid-template-columns: repeat(5, minmax(0, 1fr));

    :deep(.el-radio-button__inner) {
      width: 100%;
      padding: 10px 8px;
    }
  }

  &__needs-list {
    flex-wrap: wrap;

    :deep(.el-checkbox) {
      margin-right: 0;
    }
  }

  &__footer {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }
}

:global(.tenant-profile-completion-dialog) {
  border-radius: 14px;
}

:global(.tenant-profile-completion-dialog .el-dialog__header) {
  height: 56px;
  box-sizing: border-box;
  margin-right: 0;
  padding: 15px 40px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

:global(.tenant-profile-completion-dialog .el-dialog__title) {
  color: var(--el-text-color-primary);
  font-size: 18px;
  font-weight: 650;
  line-height: 26px;
}

:global(.tenant-profile-completion-dialog .el-dialog__headerbtn) {
  top: 12px;
  right: 24px;
  width: 32px;
  height: 32px;
}

:global(.tenant-profile-completion-dialog .el-dialog__close) {
  width: 22px;
  height: 22px;
}

:global(.tenant-profile-completion-dialog .el-dialog__body) {
  padding: 28px 36px 0;
}

:global(.tenant-profile-completion-dialog .el-dialog__footer) {
  padding: 16px 36px 20px;
  border-top: 1px solid var(--el-border-color-lighter);
}

@media (max-width: 640px) {
  .tenant-profile-completion-dialog {
    &__form {
      :deep(.el-form-item) {
        margin-bottom: 22px;
      }
    }

    &__size-list {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 8px;
    }
  }

  :global(.tenant-profile-completion-dialog .el-dialog__header) {
    height: 52px;
    padding: 13px 24px;
  }

  :global(.tenant-profile-completion-dialog .el-dialog__body) {
    padding: 22px 20px 0;
  }

  :global(.tenant-profile-completion-dialog .el-dialog__footer) {
    padding: 14px 20px 18px;
  }
}
</style>
