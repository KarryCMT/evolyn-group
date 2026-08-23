<script setup lang="ts">
import { RiCloseFill, RiFileList3Fill, RiSearchFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed } from 'vue';
import CrossAppApplicationList from './CrossAppApplicationList.vue';
import CrossAppFormTree from './CrossAppFormTree.vue';
import { useCrossAppSelection } from './useCrossAppSelection';

defineOptions({ name: 'CrossAppSetting' });

const {
  activeApplication,
  activeApplicationId,
  filteredApplications,
  hasUnsavedChanges,
  keyword,
  removeSelectedForm,
  save,
  selectApplication,
  selectedFormIds,
  selectedForms,
  toggleForm,
  toggleGroup,
} = useCrossAppSelection();

const selectedCountLabel = computed(() => `已选择 ${selectedForms.value.length} 张表单`);
const activeApplicationFormCount = computed(
  () =>
    activeApplication.value?.groups.reduce((count, group) => count + group.forms.length, 0) ?? 0,
);

function removeForm(id: string) {
  removeSelectedForm(id);
}

function saveSelection() {
  save();
  ElMessage.success('跨应用选择已保存（仅本地演示）');
}

function handleFormToggle(id: string) {
  toggleForm(id);
}
</script>

<template>
  <section class="cross-app-setting">
    <header class="cross-app-setting__header">
      <div>
        <h1 class="cross-app-setting__title">跨应用</h1>
        <p class="cross-app-setting__description">选择当前应用可调用的其他应用表单</p>
      </div>
      <el-button type="primary" class="cross-app-setting__save" @click="saveSelection">
        保存
      </el-button>
    </header>

    <div class="cross-app-setting__body">
      <section class="cross-app-setting__selection" aria-labelledby="cross-app-selected-title">
        <div class="cross-app-setting__selection-heading">
          <span id="cross-app-selected-title">已选择的表单</span>
          <span class="cross-app-setting__selection-count">{{ selectedCountLabel }}</span>
          <span v-if="hasUnsavedChanges" class="cross-app-setting__dirty">未保存</span>
        </div>
        <div v-if="selectedForms.length" class="cross-app-setting__chips">
          <span
            v-for="form in selectedForms"
            :key="form.id"
            class="cross-app-setting__chip"
            :title="`${form.applicationName} · ${form.name}`"
          >
            <RiFileList3Fill class="cross-app-setting__chip-icon" aria-hidden="true" />
            <span>{{ form.name }}</span>
            <button
              class="cross-app-setting__chip-remove"
              type="button"
              :aria-label="`移除${form.name}`"
              @click="removeForm(form.id)"
            >
              <RiCloseFill aria-hidden="true" />
            </button>
          </span>
        </div>
        <p v-else class="cross-app-setting__empty-selection">暂未选择可调用的表单</p>
      </section>

      <div class="cross-app-setting__search">
        <RiSearchFill aria-hidden="true" />
        <input
          v-model="keyword"
          type="search"
          placeholder="搜索应用、分组或表单"
          aria-label="搜索"
        />
      </div>

      <div class="cross-app-setting__picker">
        <aside class="cross-app-setting__applications" aria-label="可调用应用">
          <CrossAppApplicationList
            :applications="filteredApplications"
            :active-application-id="activeApplication?.id ?? activeApplicationId"
            :selected-form-ids="selectedFormIds"
            @select="selectApplication"
          />
        </aside>
        <section class="cross-app-setting__tree" aria-label="应用表单列表">
          <header v-if="activeApplication" class="cross-app-setting__tree-heading">
            <strong>{{ activeApplication.name }}</strong>
            <span>{{ activeApplicationFormCount }} 张表单</span>
          </header>
          <CrossAppFormTree
            v-if="activeApplication"
            :groups="activeApplication.groups"
            :selected-form-ids="selectedFormIds"
            @toggle-form="handleFormToggle"
            @toggle-group="toggleGroup"
          />
          <div v-else class="cross-app-setting__no-results">没有匹配的应用或表单</div>
        </section>
      </div>
    </div>
  </section>
</template>

<style scoped lang="scss">
.cross-app-setting {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  // RouterView 作为设置容器的 flex 子项时，占满可用横向空间，避免右侧遗留空白。
  flex: 1;
  flex-direction: column;
  background: var(--el-bg-color);

  &__header {
    display: flex;
    min-height: 74px;
    padding: 0 28px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    align-items: center;
    justify-content: space-between;
    gap: 20px;
  }

  &__title,
  &__description {
    margin: 0;
  }

  &__title {
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 650;
    line-height: 28px;
  }

  &__description {
    margin-top: 2px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }

  &__save {
    min-width: 76px;
    height: 36px;
    font-weight: 600;
  }

  &__body {
    display: flex;
    min-height: 0;
    padding: 22px 28px 28px;
    overflow: hidden;
    flex: 1;
    flex-direction: column;
    gap: 14px;
  }

  &__selection {
    min-height: 88px;
    padding: 12px 14px;
    border: 1px dashed var(--el-border-color);
    border-radius: 8px;
    background: var(--el-fill-color-blank);
  }

  &__selection-heading {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--el-text-color-regular);
    font-size: 13px;
    line-height: 20px;
  }

  &__selection-count {
    color: var(--el-text-color-secondary);
  }

  &__dirty {
    padding: 0 6px;
    border-radius: 3px;
    color: var(--el-color-warning);
    background: var(--el-color-warning-light-9);
    font-size: 11px;
    line-height: 18px;
  }

  &__chips {
    display: flex;
    margin-top: 9px;
    flex-wrap: wrap;
    gap: 8px;
  }

  &__chip {
    display: inline-flex;
    min-width: 0;
    height: 32px;
    padding: 0 4px 0 8px;
    border-radius: 4px;
    align-items: center;
    gap: 6px;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    font-size: 13px;
    line-height: 20px;
  }

  &__chip-icon {
    width: 15px;
    height: 15px;
    color: var(--el-color-primary);
  }

  &__chip-remove {
    display: inline-flex;
    width: 24px;
    height: 24px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    background: transparent;

    &:hover {
      color: var(--el-text-color-primary);
      background: var(--el-fill-color);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -1px;
    }

    svg {
      width: 16px;
      height: 16px;
    }
  }

  &__empty-selection {
    margin: 12px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }

  &__search {
    display: flex;
    height: 40px;
    padding: 0 12px;
    border-radius: 6px;
    align-items: center;
    gap: 8px;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);

    svg {
      width: 18px;
      height: 18px;
      flex: 0 0 auto;
    }

    input {
      width: 100%;
      min-width: 0;
      padding: 0;
      border: 0;
      outline: 0;
      color: var(--el-text-color-primary);
      background: transparent;
      font: inherit;
      font-size: 14px;

      &::placeholder {
        color: var(--el-text-color-placeholder);
      }
    }
  }

  &__picker {
    display: grid;
    min-height: 0;
    flex: 1;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    grid-template-columns: minmax(220px, 1fr) minmax(440px, 1.25fr);
    overflow: hidden;
  }

  &__applications {
    display: flex;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    border-right: 1px solid var(--el-border-color-lighter);
  }

  &__tree {
    display: flex;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    flex-direction: column;
  }

  &__tree-heading {
    display: flex;
    height: 50px;
    padding: 0 28px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    align-items: center;
    justify-content: space-between;
    gap: 12px;

    strong {
      color: var(--el-text-color-primary);
      font-size: 14px;
      line-height: 20px;
    }

    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
      line-height: 18px;
    }
  }

  &__no-results {
    display: grid;
    min-height: 100%;
    place-items: center;
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }
}

@media (max-width: 860px) {
  .cross-app-setting {
    &__body {
      padding: 16px;
    }

    &__header {
      padding: 0 16px;
    }

    &__picker {
      min-height: 0;
      grid-template-columns: minmax(180px, 0.75fr) minmax(360px, 1.25fr);
    }
  }
}
</style>
