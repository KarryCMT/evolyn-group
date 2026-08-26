<script setup lang="ts">
import { RiArrowDownSFill, RiArrowUpSFill, RiCloseFill, RiInboxArchiveFill } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import type {
  OrganizationMember,
  OrganizationRole,
  WorkHandoverSelection,
} from './organization.types';

defineOptions({ name: 'WorkHandoverDialog' });

type HandoverCategoryID =
  | 'roles'
  | 'pending-tasks'
  | 'process-owners'
  | 'form-configurations'
  | 'knowledge-permissions';

interface HandoverCategory {
  id: HandoverCategoryID;
  group: '系统' | '协同应用';
  label: string;
  emptyText: string;
}

const props = defineProps<{
  /** 当前被交接工作的成员。 */
  member: OrganizationMember | null;
  /** 已创建角色是当前组织页唯一可读取的可交接资源。 */
  roles: OrganizationRole[];
}>();

const emit = defineEmits<{
  chooseRecipient: [selection: WorkHandoverSelection];
}>();

const open = defineModel<boolean>({ default: false });

const categories: HandoverCategory[] = [
  {
    id: 'roles',
    group: '系统',
    label: '角色',
    emptyText: '此分类下暂无可交接的角色',
  },
  {
    id: 'pending-tasks',
    group: '协同应用',
    label: '待办任务',
    emptyText: '此分类下暂无工作',
  },
  {
    id: 'process-owners',
    group: '协同应用',
    label: '流程负责人',
    emptyText: '此分类下暂无工作',
  },
  {
    id: 'form-configurations',
    group: '协同应用',
    label: '表单/仪表盘配置',
    emptyText: '此分类下暂无工作',
  },
  {
    id: 'knowledge-permissions',
    group: '协同应用',
    label: '知识库权限',
    emptyText: '此分类下暂无工作',
  },
];

const activeCategoryID = shallowRef<HandoverCategoryID>('roles');
const selectedCategoryIDs = shallowRef<HandoverCategoryID[]>([]);
const selectedRoleIDs = shallowRef<string[]>([]);
const expandedRoleSection = shallowRef(true);

const groupedCategories = computed(() =>
  (['系统', '协同应用'] as const).map((group) => ({
    group,
    items: categories.filter((category) => category.group === group),
  })),
);
const activeCategory = computed(
  () => categories.find((category) => category.id === activeCategoryID.value) ?? categories[0],
);
const allSelected = computed(
  () => selectedCategoryIDs.value.length === categories.length && categories.length > 0,
);
const canChooseRecipient = computed(() => selectedCategoryIDs.value.length > 0);

function resetDraft() {
  activeCategoryID.value = 'roles';
  selectedCategoryIDs.value = categories.map((category) => category.id);
  selectedRoleIDs.value = props.roles.map((role) => role.id);
  expandedRoleSection.value = true;
}

function isCategorySelected(categoryID: HandoverCategoryID) {
  return selectedCategoryIDs.value.includes(categoryID);
}

function selectCategory(categoryID: HandoverCategoryID) {
  activeCategoryID.value = categoryID;
}

function toggleAll(checked: string | number | boolean) {
  selectedCategoryIDs.value = checked ? categories.map((category) => category.id) : [];
}

function toggleCategory(categoryID: HandoverCategoryID, checked: string | number | boolean) {
  selectedCategoryIDs.value = checked
    ? [...new Set([...selectedCategoryIDs.value, categoryID])]
    : selectedCategoryIDs.value.filter((id) => id !== categoryID);
}

function toggleRole(roleID: string, checked: string | number | boolean) {
  selectedRoleIDs.value = checked
    ? [...new Set([...selectedRoleIDs.value, roleID])]
    : selectedRoleIDs.value.filter((id) => id !== roleID);
}

function closeDrawer() {
  open.value = false;
}

function chooseRecipient() {
  if (!canChooseRecipient.value) return;
  emit('chooseRecipient', {
    categoryIds: [...selectedCategoryIDs.value],
    roleIds: [...selectedRoleIDs.value],
  });
}

watch(
  open,
  (visible) => {
    if (visible) resetDraft();
  },
  { immediate: true },
);
</script>

<template>
  <el-drawer
    v-model="open"
    :with-header="false"
    :append-to-body="true"
    direction="rtl"
    size="100%"
    class="work-handover-dialog"
    modal-class="work-handover-dialog__modal"
  >
    <section class="work-handover-dialog__surface" aria-label="交接工作">
      <header class="work-handover-dialog__header">
        <h2 class="work-handover-dialog__title">交接工作</h2>
        <button
          class="work-handover-dialog__close"
          type="button"
          aria-label="关闭交接工作"
          @click="closeDrawer"
        >
          <RiCloseFill aria-hidden="true" />
        </button>
      </header>

      <main class="work-handover-dialog__main">
        <section class="work-handover-dialog__workspace" aria-label="交接内容选择">
          <header class="work-handover-dialog__summary">
            <strong>{{ props.member?.name ?? '该成员' }}</strong>
            有以下工作，可根据需要转交给其他成员
          </header>

          <div class="work-handover-dialog__content">
            <aside class="work-handover-dialog__categories" aria-label="交接工作分类">
              <button
                class="work-handover-dialog__category work-handover-dialog__category--all"
                type="button"
                @click="toggleAll(!allSelected)"
              >
                <el-checkbox
                  :model-value="allSelected"
                  @click.stop
                  @update:model-value="toggleAll($event)"
                />
                <span>全选</span>
              </button>

              <section
                v-for="group in groupedCategories"
                :key="group.group"
                class="work-handover-dialog__category-group"
              >
                <h3 class="work-handover-dialog__category-group-title">{{ group.group }}</h3>
                <button
                  v-for="category in group.items"
                  :key="category.id"
                  class="work-handover-dialog__category"
                  :class="{
                    'work-handover-dialog__category--active': activeCategoryID === category.id,
                  }"
                  type="button"
                  @click="selectCategory(category.id)"
                >
                  <el-checkbox
                    :model-value="isCategorySelected(category.id)"
                    @click.stop
                    @update:model-value="toggleCategory(category.id, $event)"
                  />
                  <span>{{ category.label }}</span>
                </button>
              </section>
            </aside>

            <section class="work-handover-dialog__details" :aria-label="activeCategory.label">
              <template v-if="activeCategory.id === 'roles'">
                <p class="work-handover-dialog__details-note">
                  支持转交已创建的角色；其他角色配置请在通讯录中调整。
                </p>
                <el-scrollbar class="work-handover-dialog__details-scrollbar">
                  <section v-if="props.roles.length" class="work-handover-dialog__role-section">
                    <button
                      class="work-handover-dialog__role-section-header"
                      type="button"
                      @click="expandedRoleSection = !expandedRoleSection"
                    >
                      <span>可交接角色</span>
                      <component
                        :is="expandedRoleSection ? RiArrowUpSFill : RiArrowDownSFill"
                        aria-hidden="true"
                      />
                    </button>
                    <div v-show="expandedRoleSection" class="work-handover-dialog__role-list">
                      <label
                        v-for="role in props.roles"
                        :key="role.id"
                        class="work-handover-dialog__role-item"
                      >
                        <el-checkbox
                          :model-value="selectedRoleIDs.includes(role.id)"
                          @update:model-value="toggleRole(role.id, $event)"
                        />
                        <span>{{ role.name }}</span>
                      </label>
                    </div>
                  </section>
                  <div v-else class="work-handover-dialog__empty-state">
                    <RiInboxArchiveFill aria-hidden="true" />
                    <p>{{ activeCategory.emptyText }}</p>
                  </div>
                </el-scrollbar>
              </template>

              <div v-else class="work-handover-dialog__empty-state">
                <RiInboxArchiveFill aria-hidden="true" />
                <p>{{ activeCategory.emptyText }}</p>
              </div>
            </section>
          </div>

          <footer class="work-handover-dialog__footer">
            <el-button type="primary" :disabled="!canChooseRecipient" @click="chooseRecipient"
              >转交</el-button
            >
          </footer>
        </section>
      </main>
    </section>
  </el-drawer>
</template>

<style scoped lang="scss">
:global(.work-handover-dialog .el-drawer__body) {
  padding: 0;
}

:global(.work-handover-dialog__modal) {
  background: rgb(0 0 0 / 48%);
}

.work-handover-dialog__surface {
  display: flex;
  height: 100%;
  min-width: 960px;
  flex-direction: column;
  background: var(--el-bg-color-page);
}

.work-handover-dialog__header {
  position: relative;
  display: flex;
  height: 56px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  align-items: center;
  justify-content: center;
  background: var(--el-bg-color);
}

.work-handover-dialog__title {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 18px;
  font-weight: 600;
  line-height: 26px;
}

.work-handover-dialog__close {
  position: absolute;
  top: 12px;
  right: 24px;
  display: inline-flex;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 0;
  border-radius: var(--el-border-radius-base);
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-regular);
  background: transparent;
  cursor: pointer;
}

.work-handover-dialog__close:hover {
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
}

.work-handover-dialog__close svg {
  width: 22px;
  height: 22px;
}

.work-handover-dialog__main {
  display: flex;
  min-height: 0;
  padding: 22px;
  flex: 1;
  align-items: center;
  justify-content: center;
}

.work-handover-dialog__workspace {
  display: flex;
  width: min(920px, 100%);
  min-width: 860px;
  height: min(782px, calc(100vh - 100px));
  min-height: 520px;
  flex-direction: column;
  border-radius: var(--el-border-radius-base);
  overflow: hidden;
  background: var(--el-bg-color);
  box-shadow: var(--el-box-shadow-light);
}

.work-handover-dialog__summary {
  display: flex;
  min-height: 58px;
  padding: 0 22px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  align-items: center;
  color: var(--el-text-color-primary);
  font-size: 16px;
}

.work-handover-dialog__summary strong {
  margin-right: 4px;
  font-weight: 600;
}

.work-handover-dialog__content {
  display: flex;
  min-height: 0;
  flex: 1;
}

.work-handover-dialog__categories {
  width: 180px;
  border-right: 1px solid var(--el-border-color-lighter);
  flex: 0 0 180px;
}

.work-handover-dialog__category-group {
  padding: 14px 0 4px;
}

.work-handover-dialog__category-group-title {
  margin: 0;
  padding: 0 14px 8px;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  font-weight: 400;
}

.work-handover-dialog__category {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  min-height: 40px;
  padding: 0 14px;
  border: 0;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.work-handover-dialog__category:hover,
.work-handover-dialog__category--active {
  background: var(--el-fill-color-light);
}

.work-handover-dialog__category--all {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.work-handover-dialog__details {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.work-handover-dialog__details-note {
  margin: 18px 20px 12px;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  line-height: 22px;
}

.work-handover-dialog__details-scrollbar {
  min-height: 0;
  margin: 0 10px 12px 20px;
  flex: 1;
}

.work-handover-dialog__role-section {
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
}

.work-handover-dialog__role-section-header {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  min-height: 42px;
  padding: 0 14px;
  border: 0;
  align-items: center;
  justify-content: space-between;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  cursor: pointer;
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  text-align: left;
}

.work-handover-dialog__role-section-header:hover {
  background: var(--el-fill-color);
}

.work-handover-dialog__role-section-header svg {
  width: 18px;
  height: 18px;
}

.work-handover-dialog__role-list {
  display: grid;
  padding: 4px 0;
}

.work-handover-dialog__role-item {
  display: flex;
  min-height: 42px;
  padding: 0 14px;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  font-size: 14px;
}

.work-handover-dialog__role-item:hover {
  background: var(--el-fill-color-light);
}

.work-handover-dialog__empty-state {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
}

.work-handover-dialog__empty-state svg {
  width: 60px;
  height: 60px;
  margin-bottom: 14px;
  color: var(--el-color-primary-light-5);
}

.work-handover-dialog__empty-state p {
  margin: 0;
  font-size: 14px;
}

.work-handover-dialog__footer {
  display: flex;
  min-height: 64px;
  border-top: 1px solid var(--el-border-color-lighter);
  align-items: center;
  justify-content: center;
}

.work-handover-dialog__footer :deep(.el-button) {
  min-width: 58px;
}

@media (max-width: 768px) {
  .work-handover-dialog__surface {
    min-width: 0;
  }

  .work-handover-dialog__header {
    height: 52px;
  }

  .work-handover-dialog__close {
    top: 10px;
    right: 12px;
  }

  .work-handover-dialog__main {
    padding: 0;
  }

  .work-handover-dialog__workspace {
    width: 100%;
    min-width: 0;
    height: 100%;
    min-height: 0;
    border-radius: 0;
    box-shadow: none;
  }

  .work-handover-dialog__categories {
    width: 148px;
    flex-basis: 148px;
  }

  .work-handover-dialog__summary {
    padding: 0 14px;
    font-size: 14px;
  }
}
</style>
