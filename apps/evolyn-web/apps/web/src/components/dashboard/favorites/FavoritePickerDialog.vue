<script setup lang="ts">
import type { FavoriteApplication } from './favoriteCatalog';
import { ArrowDown, ArrowRight, Check, Close, Search } from '@element-plus/icons-vue';
import { computed, shallowRef, watch } from 'vue';
import { favoriteApplicationCatalog } from './favoriteCatalog';

defineOptions({ name: 'FavoritePickerDialog' });

const props = defineProps<{
  selectedIds: string[];
}>();

const emit = defineEmits<{
  confirm: [ids: string[]];
}>();

const visible = defineModel<boolean>({ default: false });

const searchText = shallowRef('');
const draftSelectedIds = shallowRef<string[]>([]);
const expandedIds = shallowRef(new Set(['sample-app']));

// 每次打开均以已保存收藏为基准创建草稿，取消不会修改工作台和收藏面板。
watch(
  () => visible.value,
  (isVisible) => {
    if (!isVisible) return;
    searchText.value = '';
    draftSelectedIds.value = [...props.selectedIds];
  },
);

const visibleApplications = computed(() =>
  filterApplications(favoriteApplicationCatalog, searchText.value),
);

function filterApplications(
  applications: FavoriteApplication[],
  keyword: string,
): FavoriteApplication[] {
  const normalizedKeyword = keyword.trim().toLocaleLowerCase();
  if (!normalizedKeyword) return applications;

  return applications.flatMap((application) => {
    const matchingChildren = filterApplications(application.children ?? [], normalizedKeyword);
    const isMatched = application.label.toLocaleLowerCase().includes(normalizedKeyword);
    return isMatched || matchingChildren.length
      ? [{ ...application, children: matchingChildren }]
      : [];
  });
}

function isChecked(id: string) {
  return draftSelectedIds.value.includes(id);
}

function toggleSelection(id: string) {
  draftSelectedIds.value = isChecked(id)
    ? draftSelectedIds.value.filter((selectedId) => selectedId !== id)
    : [...draftSelectedIds.value, id];
}

function toggleExpanded(id: string) {
  const nextExpandedIds = new Set(expandedIds.value);
  nextExpandedIds.has(id) ? nextExpandedIds.delete(id) : nextExpandedIds.add(id);
  expandedIds.value = nextExpandedIds;
}

function isExpanded(application: FavoriteApplication) {
  return Boolean(searchText.value.trim()) || expandedIds.value.has(application.id);
}

function confirm() {
  emit('confirm', draftSelectedIds.value);
  visible.value = false;
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="favorite-picker-dialog"
    width="630px"
    top="8vh"
    :show-close="false"
    :close-on-click-modal="false"
    append-to-body
  >
    <template #header>
      <header class="favorite-picker-dialog__header">
        <h2 class="favorite-picker-dialog__heading">添加应用</h2>
        <el-button
          text
          class="favorite-picker-dialog__close"
          :icon="Close"
          aria-label="关闭"
          @click="visible = false"
        />
      </header>
    </template>

    <div class="favorite-picker-dialog__body">
      <el-input
        v-model="searchText"
        class="favorite-picker-dialog__search"
        :prefix-icon="Search"
        placeholder="搜索应用"
        clearable
      />

      <div class="favorite-picker-dialog__list" role="tree" aria-label="应用列表">
        <template v-for="application in visibleApplications" :key="application.id">
          <div
            class="favorite-picker-dialog__row"
            role="treeitem"
            :aria-expanded="application.children?.length ? isExpanded(application) : undefined"
          >
            <button
              v-if="application.children?.length"
              type="button"
              class="favorite-picker-dialog__expander"
              :aria-label="
                isExpanded(application) ? `收起${application.label}` : `展开${application.label}`
              "
              @click="toggleExpanded(application.id)"
            >
              <el-icon>
                <component :is="isExpanded(application) ? ArrowDown : ArrowRight" />
              </el-icon>
            </button>
            <span v-else class="favorite-picker-dialog__indent" aria-hidden="true" />
            <span
              class="favorite-picker-dialog__app-icon"
              :class="`favorite-picker-dialog__app-icon--${application.tone}`"
              aria-hidden="true"
            >
              <el-icon><component :is="application.icon" /></el-icon>
            </span>
            <span class="favorite-picker-dialog__name">{{ application.label }}</span>
            <button
              type="button"
              class="favorite-picker-dialog__checkbox"
              :class="{ 'favorite-picker-dialog__checkbox--checked': isChecked(application.id) }"
              role="checkbox"
              :aria-checked="isChecked(application.id)"
              :aria-label="`收藏${application.label}`"
              @click="toggleSelection(application.id)"
            >
              <el-icon v-if="isChecked(application.id)">
                <Check />
              </el-icon>
            </button>
          </div>

          <div
            v-if="application.children?.length && isExpanded(application)"
            class="favorite-picker-dialog__children"
            role="group"
          >
            <div
              v-for="child in application.children"
              :key="child.id"
              class="favorite-picker-dialog__row favorite-picker-dialog__row--child"
              role="treeitem"
            >
              <span class="favorite-picker-dialog__indent" aria-hidden="true" />
              <span
                class="favorite-picker-dialog__app-icon"
                :class="`favorite-picker-dialog__app-icon--${child.tone}`"
                aria-hidden="true"
              >
                <el-icon><component :is="child.icon" /></el-icon>
              </span>
              <span class="favorite-picker-dialog__name">{{ child.label }}</span>
              <button
                type="button"
                class="favorite-picker-dialog__checkbox"
                :class="{ 'favorite-picker-dialog__checkbox--checked': isChecked(child.id) }"
                role="checkbox"
                :aria-checked="isChecked(child.id)"
                :aria-label="`收藏${child.label}`"
                @click="toggleSelection(child.id)"
              >
                <el-icon v-if="isChecked(child.id)">
                  <Check />
                </el-icon>
              </button>
            </div>
          </div>
        </template>
      </div>
    </div>

    <template #footer>
      <div class="favorite-picker-dialog__footer">
        <el-button size="large" @click="visible = false"> 取消 </el-button>
        <el-button type="primary" size="large" @click="confirm"> 确定 </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style lang="scss">
.favorite-picker-dialog.el-dialog {
  display: flex;
  flex-direction: column;
  height: 600px;
  margin-bottom: 0;
  overflow: hidden;
  border-radius: var(--el-border-radius-round);

  /* 选择器位于 body 弹层树中，不能依赖工作台容器的主题变量。 */
  --el-color-primary: #1677ff;
  --el-color-primary-light-3: #5ca0ff;
  --el-color-primary-light-7: #b9d6ff;
  --el-color-primary-light-9: #e8f1ff;
}

.favorite-picker-dialog .el-dialog__header {
  flex: 0 0 auto;
  padding: 0;
  margin: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.favorite-picker-dialog .el-dialog__body {
  flex: 1;
  min-height: 0;
  padding: var(--el-space-3xl) var(--el-space-3xl);
  overflow: hidden;
}

.favorite-picker-dialog .el-dialog__footer {
  flex: 0 0 auto;
  padding: var(--el-space-xl) var(--el-space-3xl);
  border-top: 1px solid var(--el-border-color-lighter);
}

.favorite-picker-dialog__header {
  position: relative;
  display: flex;
  align-items: center;
  height: 56px;
  padding: 0 var(--el-space-3xl);
}

.favorite-picker-dialog__heading {
  margin: 0;
  font-size: var(--el-font-size-extra-large);
  font-weight: 650;
  color: var(--el-text-color-primary);
}

.favorite-picker-dialog__close.el-button {
  position: absolute;
  top: 10px;
  right: 14px;
  width: 32px;
  height: 32px;
  padding: 0;
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-extra-large);
}

.favorite-picker-dialog__body {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.favorite-picker-dialog__search.el-input {
  flex: 0 0 auto;
}

.favorite-picker-dialog__search .el-input__wrapper {
  min-height: 48px;
  padding: 0 var(--el-space-lg);
  background: var(--el-fill-color-light);
  border-radius: var(--el-border-radius-medium);
  box-shadow: none;
}

.favorite-picker-dialog__search .el-input__inner {
  font-size: var(--el-font-size-medium);
}

.favorite-picker-dialog__search .el-input__prefix-inner {
  font-size: var(--el-font-size-extra-large);
}

.favorite-picker-dialog__list {
  flex: 1;
  min-height: 0;
  padding: var(--el-space-lg) 0;
  overflow: auto;
  scrollbar-color: var(--el-border-color) transparent;
}

.favorite-picker-dialog__row {
  display: flex;
  align-items: center;
  min-height: 48px;
  padding-right: var(--el-space-lg);
}

.favorite-picker-dialog__row--child {
  padding-left: var(--el-space-4xl);
}

.favorite-picker-dialog__expander,
.favorite-picker-dialog__indent {
  display: inline-flex;
  flex: 0 0 26px;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
}

.favorite-picker-dialog__expander {
  padding: 0;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;
}

.favorite-picker-dialog__app-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  margin-right: var(--el-space-md);
  color: var(--el-color-white);
  border-radius: var(--el-border-radius-medium);
}

.favorite-picker-dialog__app-icon--blue {
  background: #4b8cf7;
}
.favorite-picker-dialog__app-icon--cyan {
  background: #1aaee2;
}
.favorite-picker-dialog__app-icon--green {
  background: #48b860;
}
.favorite-picker-dialog__app-icon--orange {
  background: #ff9d32;
}
.favorite-picker-dialog__app-icon--purple {
  background: #8367ee;
}
.favorite-picker-dialog__app-icon--red {
  background: #f36061;
}

.favorite-picker-dialog__name {
  min-width: 0;
  overflow: hidden;
  font-size: var(--el-font-size-medium);
  line-height: 1.4;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.favorite-picker-dialog__checkbox {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  margin-left: auto;
  color: var(--el-color-white);
  cursor: pointer;
  background: var(--el-bg-color);
  border: 2px solid var(--el-border-color-darker);
  border-radius: var(--el-border-radius-medium);
}

.favorite-picker-dialog__checkbox--checked {
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
}

.favorite-picker-dialog__checkbox .el-icon {
  font-size: var(--el-font-size-large);
}

.favorite-picker-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--el-space-lg);
}

.favorite-picker-dialog__footer .el-button {
  min-width: 88px;
  height: 44px;
  margin: 0;
  font-size: var(--el-font-size-medium);
  border-radius: var(--el-border-radius-medium);
}

@media (max-width: 720px) {
  .favorite-picker-dialog.el-dialog {
    width: 100vw !important;
    height: 100vh;
    top: 0;
    border-radius: 0;
  }
  .favorite-picker-dialog .el-dialog__body {
    padding: var(--el-space-3xl) var(--el-space-2xl);
  }
  .favorite-picker-dialog .el-dialog__footer {
    padding: var(--el-space-xl) var(--el-space-2xl);
  }
  .favorite-picker-dialog__header {
    height: 76px;
    padding: 0 var(--el-space-2xl);
  }
  .favorite-picker-dialog__heading {
    font-size: var(--el-font-size-extra-large);
  }
  .favorite-picker-dialog__name {
    font-size: var(--el-font-size-large);
  }
  .favorite-picker-dialog__footer .el-button {
    height: 46px;
    font-size: var(--el-font-size-medium);
  }
}
</style>
