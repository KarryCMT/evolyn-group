<script setup lang="ts">
import type { BusinessDashboardDocument } from '@evolyn.do/dashboard';
import {
  BusinessDashboardRenderer,
  normalizeBusinessDashboardDocument,
} from '@evolyn.do/dashboard';
import { computed, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getDashboard } from '~/api/dashboard';

defineOptions({ name: 'DashboardPreviewPage' });

const route = useRoute();
const router = useRouter();
const appCode = computed(() => String(route.params.appCode ?? ''));
const dashboardCode = computed(() => String(route.params.dashboardCode ?? ''));
const requestedRevision = computed(() => Number(route.query.revision ?? 0));
const status = shallowRef<'loading' | 'ready' | 'error'>('loading');
const name = shallowRef('仪表盘预览');
const previewDocument = shallowRef<BusinessDashboardDocument | null>(null);
const errorMessage = shallowRef('');

watch(
  [dashboardCode, requestedRevision],
  async ([code, revision]) => {
    status.value = 'loading';
    errorMessage.value = '';
    try {
      const detail = await getDashboard(code);
      if (revision !== detail.draftRevision) {
        throw new Error('该草稿修订已不是服务端当前版本，请返回设计页重新预览。');
      }
      const normalized = normalizeBusinessDashboardDocument(detail.draft);
      if (!normalized.document) throw new Error('草稿内容无法渲染。');
      name.value = detail.name;
      previewDocument.value = normalized.document;
      document.title = `${detail.name} - 草稿预览`;
      status.value = 'ready';
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : '预览加载失败。';
      status.value = 'error';
    }
  },
  { immediate: true },
);

function backToDesign() {
  void router.push({
    name: 'dashboard-design',
    params: { appCode: appCode.value, dashboardCode: dashboardCode.value },
  });
}
</script>

<template>
  <main class="preview-page">
    <header class="preview-page__header">
      <el-button text @click="backToDesign">
        返回设计
      </el-button>
      <div>
        <span>DRAFT PREVIEW</span><strong>{{ name }}</strong>
      </div>
      <code>REV {{ requestedRevision }}</code>
    </header>
    <section v-if="status === 'loading'" v-loading="true" class="preview-page__body" />
    <el-result
      v-else-if="status === 'error'"
      class="preview-page__body"
      icon="error"
      title="无法预览草稿"
      :sub-title="errorMessage"
    >
      <template #extra>
        <el-button type="primary" @click="backToDesign">
          返回设计页
        </el-button>
      </template>
    </el-result>
    <BusinessDashboardRenderer
      v-else-if="previewDocument"
      class="preview-page__body"
      :document="previewDocument"
    />
  </main>
</template>

<style scoped>
.preview-page {
  display: flex;
  min-height: 100vh;
  flex-direction: column;
  background: #eef2f5;
}
.preview-page__header {
  display: grid;
  min-height: 64px;
  padding: 0 22px;
  align-items: center;
  grid-template-columns: 1fr auto 1fr;
  background: #fff;
  border-bottom: 1px solid rgba(23, 32, 51, 0.08);
}
.preview-page__header > div {
  display: flex;
  align-items: center;
  gap: 11px;
}
.preview-page__header span {
  color: #0f8f84;
  font:
    800 9px/1 ui-monospace,
    monospace;
  letter-spacing: 0.14em;
}
.preview-page__header strong {
  font-size: 14px;
}
.preview-page__header code {
  justify-self: end;
  color: #7c8799;
  font-size: 11px;
}
.preview-page__body {
  flex: 1;
  min-height: calc(100vh - 64px);
}
</style>
