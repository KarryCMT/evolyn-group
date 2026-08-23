<script setup lang="ts">
import {
  RiArrowLeftLine,
  RiBookmark3Fill,
  RiBriefcase4Fill,
  RiCheckboxCircleFill,
  RiContactsBook3Fill,
  RiPieChart2Fill,
} from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, markRaw, type Component } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import ApplicationEmptyState from '~/components/application/runtime/ApplicationEmptyState.vue';
import type { ApplicationAssetStarter } from '~/components/application/runtime/applicationAssetCatalog';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import { useApplicationHome } from '~/composables/useApplicationHome';
import type { ApplicationIcon } from '~/types';

defineOptions({ name: 'ApplicationHomePage' });

const route = useRoute();
const router = useRouter();
const appCode = computed(() => String(route.params.appCode ?? ''));
const { application, applicationName, errorMessage, reload, status } = useApplicationHome(appCode);

const iconByKey: Record<ApplicationIcon, Component> = {
  bookmark: markRaw(RiBookmark3Fill),
  briefcase: markRaw(RiBriefcase4Fill),
  contacts: markRaw(RiContactsBook3Fill),
  chart: markRaw(RiPieChart2Fill),
  check: markRaw(RiCheckboxCircleFill),
};

const applicationIcon = computed(() => iconByKey[application.value?.icon ?? 'bookmark']);

function returnToDashboard() {
  void router.push({ name: 'dashboard' });
}

function notifyAssetUnavailable(starter: ApplicationAssetStarter) {
  const labels: Record<ApplicationAssetStarter['type'], string> = {
    'workflow-form': '流程表单',
    form: '表单',
    dashboard: '仪表盘',
  };
  ElMessage.info(`${labels[starter.type]}能力将在后续版本接入`);
}

function notifyManagementUnavailable() {
  ElMessage.info('应用后台正在建设中');
}
</script>

<template>
  <div class="application-home-page">
    <TopNavigation :title="applicationName" :show-default-navigation="false" surface="surface">
      <template #leading>
        <button
          class="application-home-page__back"
          type="button"
          aria-label="返回工作台"
          @click="returnToDashboard"
        >
          <RiArrowLeftLine />
        </button>
      </template>
      <template #title>
        <span class="application-home-page__title">
          <span class="application-home-page__icon" aria-hidden="true">
            <component :is="applicationIcon" />
          </span>
          <strong>{{ applicationName }}</strong>
        </span>
      </template>
    </TopNavigation>

    <section v-if="status === 'loading'" v-loading="true" class="application-home-page__status" />

    <el-result
      v-else-if="status === 'not-found'"
      class="application-home-page__result"
      icon="warning"
      title="应用不存在或已不可访问"
      sub-title="请返回工作台后重新选择应用。"
    >
      <template #extra>
        <el-button type="primary" @click="returnToDashboard">返回工作台</el-button>
      </template>
    </el-result>

    <el-result
      v-else-if="status === 'error'"
      class="application-home-page__result"
      icon="error"
      title="加载应用失败"
      :sub-title="errorMessage"
    >
      <template #extra>
        <el-button type="primary" @click="reload()">重新加载</el-button>
      </template>
    </el-result>

    <!-- 当前先完成应用维度的空态；表单运行时由后续 @evolyn.do/form 包承接。 -->
    <ApplicationEmptyState
      v-else
      @select-asset="notifyAssetUnavailable"
      @open-management="notifyManagementUnavailable"
    />
  </div>
</template>

<style scoped lang="scss">
.application-home-page {
  display: flex;
  height: 100vh;
  overflow: hidden;
  flex-direction: column;
  background: var(--el-fill-color-lighter);

  &__back {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);
    font-size: 24px;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__title {
    display: inline-flex;
    min-width: 0;
    align-items: center;
    gap: 10px;
    color: var(--el-text-color-primary);

    strong {
      overflow: hidden;
      font-size: 17px;
      font-weight: 650;
      line-height: 26px;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  &__icon {
    display: inline-flex;
    width: 30px;
    height: 30px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    color: var(--el-color-white);
    background: var(--el-color-primary);
    border-radius: 7px;
    font-size: 19px;
  }

  &__status,
  &__result {
    flex: 1;
  }

  &__status {
    min-height: 0;
  }

  &__result {
    display: grid;
    place-content: center;
    background: var(--el-fill-color-lighter);
  }
}
</style>
