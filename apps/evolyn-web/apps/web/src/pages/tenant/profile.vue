<script setup lang="ts">
import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import TenantProfileBasicInfo from '~/components/tenant/profile/TenantProfileBasicInfo.vue';
import TenantProfileCompletionDialog, {
  type TenantProfileCompletion,
} from '~/components/tenant/profile/TenantProfileCompletionDialog.vue';
import TenantProfileNameEditor from '~/components/tenant/profile/TenantProfileNameEditor.vue';
import { useAuth } from '~/composables';

defineOptions({ name: 'TenantProfilePage' });

const { userInfo } = useAuth();
const editingName = shallowRef(false);
const completionVisible = shallowRef(false);
const updatedCompanyName = shallowRef<string | null>(null);
const profileCompletion = shallowRef<TenantProfileCompletion>({
  companyName: '',
  companySize: '11-50',
  industry: '互联网/软件',
  managementNeeds: ['IT项目', '任务', '合同'],
  role: 'CEO/老板',
});

const companyName = computed(() => {
  return updatedCompanyName.value ?? userInfo.value?.tenant.name ?? '未命名企业';
});

const tenantIdentifier = computed(() => {
  const tenant = userInfo.value?.tenant;
  return tenant?.code || (tenant ? String(tenant.id) : '暂未生成');
});

const companyUrl = computed(() => {
  const tenant = userInfo.value?.tenant;
  if (!tenant) return 'https://evolyn.do/portal/tenant/未生成/signin';
  return `https://evolyn.do/portal/tenant/${tenant.code || tenant.id}/signin`;
});

const completionRate = computed(() => {
  const profile = profileCompletion.value;
  const values = [profile.role, profile.companyName, profile.industry, profile.companySize];
  const completed = values.filter(Boolean).length + (profile.managementNeeds.length ? 1 : 0);
  return Math.round((completed / 5) * 100);
});

function openNameEditor() {
  editingName.value = true;
}

function saveCompanyName(name: string) {
  updatedCompanyName.value = name;
  profileCompletion.value = { ...profileCompletion.value, companyName: name };
  editingName.value = false;
  // 当前租户自助更新接口尚未提供；保留交互状态，后续可在此接入租户资料更新 API。
  ElMessage.success('企业名称已更新');
}

async function copyText(value: string, label: string) {
  try {
    await navigator.clipboard.writeText(value);
    ElMessage.success(`${label}已复制`);
  } catch {
    ElMessage.warning(`浏览器未授予剪贴板权限，请手动复制${label}`);
  }
}

function previewCompanyUrl() {
  window.open(companyUrl.value, '_blank', 'noopener,noreferrer');
}

function openCompletionDialog() {
  profileCompletion.value = {
    ...profileCompletion.value,
    companyName: profileCompletion.value.companyName || companyName.value,
    industry: userInfo.value?.tenant.config.onboarding.industry || profileCompletion.value.industry,
    managementNeeds: [
      ...(userInfo.value?.tenant.config.onboarding.managementNeeds ||
        profileCompletion.value.managementNeeds),
    ],
  };
  completionVisible.value = true;
}

function saveCompletion(profile: TenantProfileCompletion) {
  profileCompletion.value = profile;
  updatedCompanyName.value = profile.companyName;
  ElMessage.success('公司信息已完善');
}
</script>

<template>
  <section class="tenant-profile-page" aria-label="企业信息">
    <TenantProfileBasicInfo
      v-if="!editingName"
      account-mode="公共模式"
      :company-name="companyName"
      :company-url="companyUrl"
      :tenant-identifier="tenantIdentifier"
      @copy="copyText"
      @edit-name="openNameEditor"
      @preview="previewCompanyUrl"
    />

    <section
      v-else
      class="tenant-profile-page__name-edit-section"
      aria-labelledby="tenant-profile-name-title"
    >
      <h1 id="tenant-profile-name-title" class="tenant-profile-page__section-title">基础信息</h1>
      <div class="tenant-profile-page__name-editor-row">
        <span class="tenant-profile-page__name-label">企业名称</span>
        <TenantProfileNameEditor
          :model-value="companyName"
          :visible="editingName"
          @cancel="editingName = false"
          @save="saveCompanyName"
        />
      </div>
    </section>

    <section
      class="tenant-profile-page__completion"
      aria-labelledby="tenant-profile-completion-title"
    >
      <h2 id="tenant-profile-completion-title" class="tenant-profile-page__section-title">
        公司信息完善
      </h2>
      <div class="tenant-profile-page__completion-banner">
        <p>
          当前信息完整度 {{ completionRate }}%，补充公司相关信息，便于我们为你提供更加精准的服务。
        </p>
        <button
          class="tenant-profile-page__completion-action"
          type="button"
          @click="openCompletionDialog"
        >
          立即完善
        </button>
      </div>
    </section>

    <TenantProfileCompletionDialog
      v-model:visible="completionVisible"
      :initial-value="profileCompletion"
      @save="saveCompletion"
    />
  </section>
</template>

<style scoped lang="scss">
.tenant-profile-page {
  box-sizing: border-box;
  min-height: 100%;
  padding: var(--el-space-4xl) var(--el-space-3xl) var(--el-space-6xl);

  &__name-edit-section {
    min-height: 360px;
  }

  &__section-title {
    position: relative;
    margin: 0;
    padding-left: var(--el-space-xl);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-extra-large);
    font-weight: 650;
    line-height: 28px;

    &::before {
      position: absolute;
      top: 4px;
      bottom: 4px;
      left: 0;
      width: 4px;
      border-radius: var(--el-border-radius-half);
      background: var(--el-color-primary);
      content: '';
    }
  }

  &__name-editor-row {
    display: grid;
    margin-top: var(--el-space-5xl);
    grid-template-columns: 220px minmax(0, 1fr);
  }

  &__name-label {
    padding-top: 90px;
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    line-height: 24px;
  }

  &__completion {
    margin-top: var(--el-space-4xl);
    padding-top: var(--el-space-4xl);
    border-top: 1px solid var(--el-border-color-lighter);
  }

  &__completion-banner {
    display: flex;
    min-height: 88px;
    margin-top: var(--el-space-2xl);
    padding: var(--el-space-xl) var(--el-space-2xl);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    gap: var(--el-space-2xl);
    background: var(--el-fill-color-light);
  }

  &__completion-banner p {
    margin: 0;
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__completion-action {
    flex: 0 0 auto;
    padding: var(--el-space-xs) var(--el-space-sm);
    border: 0;
    border-radius: var(--el-border-radius-base);
    color: var(--el-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
    line-height: 24px;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    &:hover {
      color: var(--el-color-primary-light-3);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }
}

@media (max-width: 840px) {
  .tenant-profile-page {
    padding: var(--el-space-3xl) var(--el-space-2xl) var(--el-space-4xl);

    &__name-editor-row {
      gap: var(--el-space-md);
      grid-template-columns: 1fr;
    }

    &__name-label {
      padding-top: 0;
    }

    &__completion-banner {
      align-items: flex-start;
      flex-direction: column;
      gap: var(--el-space-md);
    }
  }
}
</style>
