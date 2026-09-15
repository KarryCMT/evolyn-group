<script setup lang="ts">
import type { UpdateAppPayload } from '~/types';
import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import { useRoute } from 'vue-router';
import { updateApp } from '~/api/apps';
import AppBasicSettingsPanel from '~/components/app/setting/AppBasicSettingsPanel.vue';
import { useAppHome } from '~/composables/useAppHome';

defineOptions({ name: 'AppSettingBasicPage' });

const route = useRoute();
const appCode = computed(() => String(route.params.appCode ?? ''));
const { app, errorMessage, reload, status } = useAppHome(appCode);
const saving = shallowRef(false);

async function updateBasicInfo(payload: UpdateAppPayload) {
  const currentApp = app.value;
  if (!currentApp || saving.value) return;

  saving.value = true;
  try {
    await updateApp(currentApp.id, payload);
    await reload();
    ElMessage.success('应用设置已保存');
  } catch {
    ElMessage.error('保存应用设置失败，请稍后重试');
  } finally {
    saving.value = false;
  }
}

async function copyAppId(value: string) {
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
</script>

<template>
  <section v-if="status === 'loading'" v-loading="true" class="app-setting-basic__status" />

  <el-result
    v-else-if="status === 'not-found'"
    class="app-setting-basic__result"
    icon="warning"
    title="应用不存在或已不可访问"
    sub-title="请返回工作台后重新选择应用。"
  />

  <el-result
    v-else-if="status === 'error'"
    class="app-setting-basic__result"
    icon="error"
    title="加载应用设置失败"
    :sub-title="errorMessage"
  >
    <template #extra>
      <el-button type="primary" @click="reload()"> 重新加载 </el-button>
    </template>
  </el-result>

  <AppBasicSettingsPanel
    v-else-if="app"
    :app="app"
    :saving="saving"
    @configure-home="notifyUnavailable"
    @configure-url="notifyUnavailable"
    @copy-id="copyAppId"
    @update="updateBasicInfo"
  />
</template>

<style scoped lang="scss">
.app-setting-basic__status,
.app-setting-basic__result {
  display: grid;
  min-height: 100%;
  place-items: center;
}
</style>
