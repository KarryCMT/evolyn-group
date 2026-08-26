<script setup lang="ts">
import {
  RiAddFill,
  RiArrowDownSFill,
  RiGroupFill,
  RiSearchFill,
  RiUserFill,
  RiUserSettingsFill,
} from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import { getAdminGroup, type AdminGroupSummaryDto } from '~/api/adminGroup';
import type { AdministratorScope } from './administrator.types';

defineOptions({ name: 'AdministratorGroupList' });

const props = defineProps<{
  scope: AdministratorScope;
  groups: AdminGroupSummaryDto[];
  /** 选中组 ID；列表为空或未选中时为 null。 */
  selectedId: number | null;
  loading?: boolean;
}>();

const emit = defineEmits<{
  select: [id: number];
  add: [];
}>();

const groupTitle = props.scope === 'system' ? '系统管理组' : '普通管理组';
const filterMenuVisible = shallowRef(false);
const groupPickerVisible = shallowRef(false);
const filterMode = shallowRef<'group' | 'member'>('group');
const keyword = shallowRef('');
const memberNamesByGroupId = shallowRef(new Map<number, string[]>());
const memberIndexLoading = shallowRef(false);

/** 下拉框只选择当前分区的自定义管理组；系统内置管理员组仍固定在上方展示。 */
const selectableGroups = computed(() => props.groups.filter((group) => !group.builtIn));
const selectedGroupName = computed(
  () => selectableGroups.value.find((group) => group.id === props.selectedId)?.name ?? '选择管理组',
);
const filteredGroups = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLocaleLowerCase();
  if (!normalizedKeyword) return selectableGroups.value;

  return selectableGroups.value.filter((group) => {
    if (filterMode.value === 'group') {
      return group.name.toLocaleLowerCase().includes(normalizedKeyword);
    }
    return (memberNamesByGroupId.value.get(group.id) ?? []).some((name) =>
      name.toLocaleLowerCase().includes(normalizedKeyword),
    );
  });
});

const pickerPlaceholder = computed(() =>
  filterMode.value === 'group' ? '搜索管理组' : '搜索成员',
);

/**
 * 管理组概要不携带成员姓名；仅切换到「筛选成员」时按需补齐成员索引，
 * 避免管理组选择的默认路径产生额外请求。
 */
async function loadMemberIndex() {
  if (
    memberIndexLoading.value ||
    memberNamesByGroupId.value.size === selectableGroups.value.length
  ) {
    return;
  }

  memberIndexLoading.value = true;
  try {
    const details = await Promise.allSettled(
      selectableGroups.value.map(async (group) => {
        const detail = await getAdminGroup(group.id);
        return [group.id, detail.members.map((member) => member.name)] as const;
      }),
    );
    // Promise.allSettled 已隔离单组失败；索引不完整时，失败组按无成员匹配处理。
    memberNamesByGroupId.value = new Map(
      details.flatMap((result) => (result.status === 'fulfilled' ? [result.value] : [])),
    );
  } finally {
    memberIndexLoading.value = false;
  }
}

function chooseFilterMode(mode: 'group' | 'member') {
  filterMode.value = mode;
  keyword.value = '';
  filterMenuVisible.value = false;
  groupPickerVisible.value = true;
  if (mode === 'member') void loadMemberIndex();
}

function selectGroup(id: number) {
  emit('select', id);
  groupPickerVisible.value = false;
  keyword.value = '';
}

watch(groupPickerVisible, (visible) => {
  if (!visible) keyword.value = '';
});
</script>

<template>
  <aside v-loading="loading" class="administrator-group-list" aria-label="管理组列表">
    <section v-if="scope === 'system'" class="administrator-group-list__built-in">
      <p class="administrator-group-list__section-title">{{ groupTitle }}</p>
      <button
        v-for="group in groups.filter((item) => item.builtIn)"
        :key="group.id"
        class="administrator-group-list__item"
        :class="{ 'administrator-group-list__item--active': group.id === selectedId }"
        type="button"
        @click="emit('select', group.id)"
      >
        <RiUserSettingsFill />
        <span>{{ group.name }}</span>
      </button>
    </section>

    <section class="administrator-group-list__custom">
      <div class="administrator-group-list__custom-heading">
        <p class="administrator-group-list__section-title">
          {{ scope === 'system' ? '通讯录管理组' : groupTitle }}
        </p>
        <button
          class="administrator-group-list__add"
          type="button"
          aria-label="添加管理组"
          @click="emit('add')"
        >
          <RiAddFill />
        </button>
      </div>
      <div class="administrator-group-list__select-wrap">
        <el-popover
          v-model:visible="filterMenuVisible"
          placement="bottom-start"
          :width="320"
          trigger="click"
          popper-class="administrator-group-list__filter-popper"
        >
          <template #reference>
            <button
              class="administrator-group-list__filter-trigger"
              type="button"
              aria-label="选择筛选方式"
              :aria-expanded="filterMenuVisible"
            >
              <RiGroupFill />
              <RiArrowDownSFill />
            </button>
          </template>
          <div class="administrator-group-list__filter-menu" role="menu" aria-label="选择筛选方式">
            <button type="button" role="menuitem" @click="chooseFilterMode('group')">
              <RiGroupFill />
              <span>筛选管理组</span>
            </button>
            <button type="button" role="menuitem" @click="chooseFilterMode('member')">
              <RiUserFill />
              <span>筛选成员</span>
            </button>
          </div>
        </el-popover>
        <el-popover
          v-model:visible="groupPickerVisible"
          placement="bottom-start"
          :width="480"
          trigger="click"
          popper-class="administrator-group-list__picker-popper"
        >
          <template #reference>
            <button
              class="administrator-group-list__select-value"
              type="button"
              aria-label="选择管理组"
              :aria-expanded="groupPickerVisible"
            >
              <span>{{ selectedGroupName }}</span>
              <RiArrowDownSFill />
            </button>
          </template>
          <div v-loading="memberIndexLoading" class="administrator-group-list__picker">
            <el-input v-model="keyword" :placeholder="pickerPlaceholder" autofocus>
              <template #prefix><RiSearchFill /></template>
            </el-input>
            <el-scrollbar max-height="250px" class="administrator-group-list__picker-results">
              <button
                v-for="group in filteredGroups"
                :key="group.id"
                type="button"
                :class="{
                  'administrator-group-list__picker-item--selected': group.id === selectedId,
                }"
                @click="selectGroup(group.id)"
              >
                <span>{{ group.name }}</span>
                <small>{{ group.memberCount }} 名成员</small>
              </button>
              <p v-if="!filteredGroups.length" class="administrator-group-list__picker-empty">
                暂无匹配的管理组
              </p>
            </el-scrollbar>
          </div>
        </el-popover>
      </div>
      <button
        v-for="group in groups.filter((item) => !item.builtIn)"
        :key="group.id"
        class="administrator-group-list__item"
        :class="{ 'administrator-group-list__item--active': group.id === selectedId }"
        type="button"
        @click="emit('select', group.id)"
      >
        <RiGroupFill />
        <span>{{ group.name }}</span>
      </button>
    </section>
  </aside>
</template>

<style scoped lang="scss">
.administrator-group-list {
  box-sizing: border-box;
  display: flex;
  min-width: 356px;
  height: 100%;
  flex-direction: column;
  padding: 30px 28px;
  border-right: 1px solid #e5e8ee;
  background: #fff;
  color: #5c6472;

  &__built-in,
  &__custom {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  &__custom {
    margin-top: 30px;
  }

  &__custom-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &__section-title {
    margin: 0;
    color: #818a98;
    font-size: 18px;
    line-height: 32px;
  }

  &__add {
    display: inline-flex;
    width: 36px;
    height: 36px;
    padding: 0;
    border: 0;
    border-radius: 9px;
    align-items: center;
    justify-content: center;
    color: #fff;
    background: var(--el-color-primary);
    cursor: pointer;

    svg {
      width: 24px;
      height: 24px;
    }

    &:hover {
      background: var(--el-color-primary-light-3);
    }
  }

  &__select-wrap {
    display: flex;
    height: 44px;
  }

  &__filter-trigger,
  &__select-value {
    display: inline-flex;
    height: 44px;
    border: 1px solid var(--el-border-color);
    align-items: center;
    background: var(--el-bg-color);
    cursor: pointer;
  }

  &__filter-trigger {
    width: 117px;
    gap: 14px;
    padding: 0 18px;
    border-radius: 7px 0 0 7px;
    color: var(--el-text-color-secondary);
  }

  &__select-value {
    flex: 1;
    justify-content: space-between;
    margin-left: -1px;
    padding: 0 12px 0 17px;
    border-radius: 0 7px 7px 0;
    color: var(--el-text-color-secondary);
    font-size: 17px;
    text-align: left;
  }

  &__filter-trigger:hover,
  &__filter-trigger[aria-expanded='true'],
  &__select-value:hover,
  &__select-value[aria-expanded='true'] {
    position: relative;
    z-index: 1;
    border-color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &__filter-trigger svg,
  &__select-value svg {
    width: 21px;
    height: 21px;
  }

  &__item {
    display: flex;
    width: 100%;
    height: 56px;
    gap: 10px;
    padding: 0 18px;
    border: 0;
    border-radius: 8px;
    align-items: center;
    color: #626b78;
    background: transparent;
    font-size: 18px;
    text-align: left;
    cursor: pointer;

    svg {
      width: 24px;
      height: 24px;
      color: #9ca5b3;
    }

    &:hover {
      background: var(--el-color-primary-light-9);
      color: var(--el-color-primary);
    }

    &--active {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
    &--active svg {
      color: var(--el-color-primary);
    }
  }
}

:global(.administrator-group-list__filter-popper.el-popper) {
  margin-top: 10px;
  padding: 7px;
  border: 0;
  border-radius: 8px;
  box-shadow: var(--el-box-shadow-light);
}

:global(.administrator-group-list__picker-popper.el-popper) {
  margin-top: 8px;
  padding: 8px;
  border: 0;
  border-radius: 10px;
  box-shadow: var(--el-box-shadow-light);
}

.administrator-group-list__filter-menu {
  display: grid;
  gap: 2px;
}

.administrator-group-list__filter-menu button {
  display: flex;
  width: 100%;
  height: 58px;
  gap: 14px;
  padding: 0 16px;
  border: 0;
  border-radius: 7px;
  align-items: center;
  color: var(--el-text-color-primary);
  background: transparent;
  font-size: 18px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}

.administrator-group-list__filter-menu button svg {
  width: 24px;
  height: 24px;
  color: var(--el-text-color-secondary);
}

.administrator-group-list__filter-menu button:hover,
.administrator-group-list__filter-menu button:focus-visible {
  outline: 0;
  background: var(--el-color-primary-light-9);
}

.administrator-group-list__picker {
  min-width: 0;
}

.administrator-group-list__picker :deep(.el-input__wrapper) {
  min-height: 48px;
  padding: 0 13px;
  box-shadow: none;
}

.administrator-group-list__picker :deep(.el-input__inner) {
  color: var(--el-text-color-primary);
  font-size: 18px;
}

.administrator-group-list__picker :deep(.el-input__prefix-inner) {
  display: inline-flex;
  color: var(--el-text-color-secondary);
}

.administrator-group-list__picker :deep(.el-input__prefix-inner svg) {
  width: 22px;
  height: 22px;
}

.administrator-group-list__picker-results {
  border-top: 1px solid var(--el-border-color-lighter);
}

.administrator-group-list__picker-results button {
  display: flex;
  width: 100%;
  min-height: 54px;
  padding: 0 16px;
  border: 0;
  border-radius: 7px;
  align-items: center;
  justify-content: space-between;
  color: var(--el-text-color-primary);
  background: transparent;
  font-size: 18px;
  text-align: left;
  cursor: pointer;
}

.administrator-group-list__picker-results button:hover,
.administrator-group-list__picker-item--selected {
  background: var(--el-color-primary-light-9) !important;
}

.administrator-group-list__picker-results small {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.administrator-group-list__picker-empty {
  margin: 0;
  padding: 24px 16px;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  text-align: center;
}
</style>
