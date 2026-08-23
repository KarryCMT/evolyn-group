<script setup lang="ts">
import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import { useRoute } from 'vue-router';
import { updateApplication } from '~/api/applications';
import ApplicationBasicSettingsPanel from '~/components/application/setting/ApplicationBasicSettingsPanel.vue';
import { useApplicationHome } from '~/composables/useApplicationHome';
import type { UpdateApplicationPayload } from '~/types';

defineOptions({ name: 'ApplicationSettingBasicPage' });

const route = useRoute();
const appCode = computed(() => String(route.params.appCode ?? ''));
const { application, errorMessage, reload, status } = useApplicationHome(appCode);
const saving = shallowRef(false);

async function updateBasicInfo(payload: UpdateApplicationPayload) {
  const currentApplication = application.value;
  if (!currentApplication || saving.value) return;

  saving.value = true;
  try {
    await updateApplication(currentApplication.id, payload);
    await reload();
    ElMessage.success('应用设置已保存');
  } catch {
    ElMessage.error('保存应用设置失败，请稍后重试');
  } finally {
    saving.value = false;
  }
}

async function copyApplicationId(value: string) {
  try {
    await navigator.clipboard.writeText(value);
    ElMessage.success('应用ID已复制');
  } catch {
    ElMessage.warning('复制失败，请手动复制应用ID');
  }
}

function notifyUnavailable() {
  ElMessage.info('该设置项将在后续版本开放');
}

function downloadIcon() {
  ElMessage.info('图标下载功能将在后续版本开放');
}
</script>

<template>
  <section v-if="status === 'loading'" v-loading="true" class="application-setting-basic__status" />

  <el-result
    v-else-if="status === 'not-found'"
    class="application-setting-basic__result"
    icon="warning"
    title="应用不存在或已不可访问"
    sub-title="请返回工作台后重新选择应用。"
  />

  <el-result
    v-else-if="status === 'error'"
    class="application-setting-basic__result"
    icon="error"
    title="加载应用设置失败"
    :sub-title="errorMessage"
  >
    <template #extra>
      <el-button type="primary" @click="reload()">重新加载</el-button>
    </template>
  </el-result>

  <ApplicationBasicSettingsPanel
    v-else-if="application"
    :application="application"
    :saving="saving"
    @configure-home="notifyUnavailable"
    @configure-url="notifyUnavailable"
    @copy-id="copyApplicationId"
    @download-icon="downloadIcon"
    @update="updateBasicInfo"
  />
</template>

<style scoped lang="scss">
.application-setting-basic__status,
.application-setting-basic__result {
  display: grid;
  min-height: 100%;
  place-items: center;
}
</style>
