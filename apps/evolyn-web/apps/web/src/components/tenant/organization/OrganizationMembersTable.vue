<script setup lang="ts">
import type {
  EvolynTableColumn,
  EvolynTableCustomRenderElement,
  EvolynTableCustomRenderObj,
} from '@evolyn.do/ui';
import { EvolynTable } from '@evolyn.do/ui';
import { RiArrowDownSFill, RiDownload2Line, RiSearch2Line } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import type { OrganizationMember, OrganizationMode } from './organization.types';

const props = defineProps<{
  mode: OrganizationMode;
  roleName: string;
  members: OrganizationMember[];
  roleNameOf: (roleId: string) => string;
  keyword: string;
  status: string;
}>();

const emit = defineEmits<{
  'update:keyword': [value: string];
  'update:status': [value: string];
  invite: [];
  addMember: [];
  export: [];
  import: [];
  edit: [member: OrganizationMember];
  remove: [member: OrganizationMember];
}>();

const ROW_HEIGHT = 64;
const currentPage = shallowRef(1);
const actionMember = shallowRef<OrganizationMember | null>(null);
const actionMenuVisible = shallowRef(false);

const tableRecords = computed(() =>
  props.members.map((member) => ({
    ...member,
    roleLabel: member.roleIds.map(props.roleNameOf).filter(Boolean).join('、'),
  })),
);

const columns = computed<EvolynTableColumn[]>(() => {
  const avatar: EvolynTableCustomRenderElement = {
    type: 'circle',
    x: 14,
    y: ROW_HEIGHT / 2,
    radius: 14,
    fill: '#f25555',
  };
  const avatarText: EvolynTableCustomRenderElement = {
    type: 'text',
    x: 14,
    y: ROW_HEIGHT / 2,
    text: '李',
    fill: '#fff',
    fontSize: 14,
    textAlign: 'center',
    textBaseline: 'middle',
  };
  const nameRender = (): EvolynTableCustomRenderObj => ({
    expectedWidth: 270,
    expectedHeight: ROW_HEIGHT,
    elements: [
      avatar,
      avatarText,
      {
        type: 'text',
        x: 36,
        y: ROW_HEIGHT / 2,
        text: '李同学',
        fill: '#394150',
        fontSize: 16,
        textBaseline: 'middle',
      },
      ...(props.mode === 'department'
        ? [
            {
              type: 'rect' as const,
              x: 96,
              y: 19,
              width: 52,
              height: 26,
              fill: '#2f80ed',
            },
            {
              type: 'text' as const,
              x: 122,
              y: ROW_HEIGHT / 2,
              text: '创建者',
              fill: '#fff',
              fontSize: 13,
              textAlign: 'center' as const,
              textBaseline: 'middle' as const,
            },
          ]
        : []),
    ],
  });
  const roleRender = (): EvolynTableCustomRenderObj => ({
    expectedWidth: 190,
    expectedHeight: ROW_HEIGHT,
    elements: props.roleName
      ? [
          { type: 'rect', x: 0, y: 17, width: 88, height: 30, fill: '#f1f5fd' },
          {
            type: 'text',
            x: 11,
            y: ROW_HEIGHT / 2,
            text: '●',
            fill: '#377ff5',
            fontSize: 13,
            textBaseline: 'middle',
          },
          {
            type: 'text',
            x: 27,
            y: ROW_HEIGHT / 2,
            text: props.roleName,
            fill: '#556071',
            fontSize: 15,
            textBaseline: 'middle',
          },
        ]
      : [],
  });
  const actionRender = (): EvolynTableCustomRenderObj => ({
    expectedWidth: 84,
    expectedHeight: ROW_HEIGHT,
    elements: [
      {
        type: 'text',
        x: 36,
        y: ROW_HEIGHT / 2,
        text: '⋮',
        fill: '#344054',
        fontSize: 25,
        textAlign: 'center',
        textBaseline: 'middle',
      },
    ],
  });

  const baseColumns: EvolynTableColumn[] = [
    { field: 'checked', title: '', width: 50, cellType: 'checkbox' },
    { field: 'name', title: '姓名', width: 270, customRender: nameRender },
  ];
  if (props.mode === 'department') {
    baseColumns.push(
      { field: 'phone', title: '手机', minWidth: 170 },
      { field: 'email', title: '邮箱', minWidth: 170 },
      { field: 'roleLabel', title: '角色', width: 190, customRender: roleRender },
    );
  } else {
    baseColumns.push(
      { field: 'department', title: '所属部门', minWidth: 260 },
      { field: 'managedDepartments', title: '分管部门', minWidth: 190 },
    );
  }
  baseColumns.push({ field: 'operation', title: '操作', width: 84, customRender: actionRender });
  return baseColumns;
});

const tableOptions = { defaultHeaderRowHeight: 58, defaultRowHeight: ROW_HEIGHT };

function openActionMenu(member: OrganizationMember) {
  actionMember.value = member;
  actionMenuVisible.value = true;
}

function onCellClick(event: unknown) {
  const payload = event as { field?: string; row?: number };
  if (payload.field !== 'operation' || payload.row === undefined || payload.row < 0) return;
  const member = props.members[payload.row];
  if (member) openActionMenu(member);
}

function editMember() {
  if (actionMember.value) emit('edit', actionMember.value);
  actionMenuVisible.value = false;
}

function removeCurrentMember() {
  if (actionMember.value) emit('remove', actionMember.value);
  actionMenuVisible.value = false;
}
</script>

<template>
  <section class="organization-members-table" aria-label="成员列表">
    <header class="organization-members-table__toolbar">
      <div class="organization-members-table__toolbar-left">
        <el-button v-if="props.mode === 'department'" type="primary" @click="emit('invite')"
          >邀请成员</el-button
        >
        <el-button v-else type="primary" @click="emit('addMember')">添加成员</el-button>
        <el-button v-if="props.mode === 'role'" @click="emit('import')">导入</el-button>
        <el-dropdown v-if="props.mode === 'role'" trigger="click">
          <el-button>导出 <RiArrowDownSFill /></el-button>
          <template #dropdown>
            <el-dropdown-menu
              ><el-dropdown-item @click="emit('export')"
                >导出当前成员</el-dropdown-item
              ></el-dropdown-menu
            >
          </template>
        </el-dropdown>
        <el-button v-else @click="emit('export')"><RiDownload2Line /> 导出</el-button>
      </div>
      <div class="organization-members-table__toolbar-right">
        <label class="organization-members-table__search">
          <RiSearch2Line />
          <input
            :value="props.keyword"
            placeholder="搜索成员"
            aria-label="搜索成员"
            @input="emit('update:keyword', ($event.target as HTMLInputElement).value)"
          />
        </label>
        <template v-if="props.mode === 'department'">
          <span class="organization-members-table__status-label">账号状态</span>
          <el-select
            :model-value="props.status"
            class="organization-members-table__status-select"
            @update:model-value="emit('update:status', $event)"
          >
            <el-option label="全部" value="全部" /><el-option
              label="已启用"
              value="已启用"
            /><el-option label="已停用" value="已停用" />
          </el-select>
        </template>
      </div>
    </header>

    <div class="organization-members-table__table-wrap">
      <EvolynTable
        class="organization-members-table__table"
        :columns="columns"
        :records="tableRecords"
        :options="tableOptions"
        empty-text="暂无成员"
        @click-cell="onCellClick"
      />
      <div
        v-if="actionMenuVisible"
        class="organization-members-table__action-menu-anchor"
        @click="actionMenuVisible = false"
      />
      <div v-if="actionMenuVisible" class="organization-members-table__action-menu-panel">
        <button type="button" @click="editMember">
          {{ props.mode === 'department' ? '编辑' : '调整分管部门' }}
        </button>
        <button v-if="props.mode === 'department'" type="button">交接工作</button>
        <button v-if="props.mode === 'department'" type="button">停用</button>
        <button
          class="organization-members-table__action-menu-panel--danger"
          type="button"
          @click="removeCurrentMember"
        >
          {{ props.mode === 'department' ? '转为离职' : '移出成员' }}
        </button>
      </div>
    </div>

    <footer class="organization-members-table__footer">
      <div class="organization-members-table__count">
        <el-select v-model="currentPage" class="organization-members-table__page-size"
          ><el-option :value="1" label="20 条/页" /></el-select
        ><span>共 {{ props.members.length }} 条</span>
      </div>
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="20"
        :total="props.members.length"
        layout="prev, pager, next"
      />
    </footer>
  </section>
</template>

<style scoped lang="scss">
.organization-members-table {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
}
.organization-members-table__toolbar {
  display: flex;
  min-height: 80px;
  padding: 0 30px;
  border-top: 1px solid var(--el-border-color-lighter);
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}
.organization-members-table__toolbar-left,
.organization-members-table__toolbar-right,
.organization-members-table__count {
  display: flex;
  align-items: center;
  gap: 14px;
}
.organization-members-table__toolbar-left :deep(.el-button) {
  height: 40px;
  font-size: 16px;
}
.organization-members-table__toolbar-left :deep(.el-button svg) {
  width: 16px;
  height: 16px;
}
.organization-members-table__search {
  display: flex;
  box-sizing: border-box;
  width: 346px;
  height: 42px;
  padding: 0 13px;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  align-items: center;
  gap: 9px;
  color: var(--el-text-color-secondary);
}
.organization-members-table__search:focus-within {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
}
.organization-members-table__search svg {
  width: 19px;
  height: 19px;
  flex: 0 0 19px;
}
.organization-members-table__search input {
  width: 100%;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 16px;
}
.organization-members-table__search input::placeholder {
  color: var(--el-text-color-placeholder);
}
.organization-members-table__status-label {
  color: var(--el-text-color-regular);
  font-size: 16px;
  white-space: nowrap;
}
.organization-members-table__status-select {
  width: 164px;
}
.organization-members-table__table-wrap {
  position: relative;
  min-height: 0;
  flex: 1;
}
.organization-members-table__table {
  height: 100%;
}
.organization-members-table__action-menu-anchor {
  position: fixed;
  z-index: 10;
  inset: 0;
  background: transparent;
}
.organization-members-table__action-menu-panel {
  position: absolute;
  z-index: 11;
  top: 125px;
  right: 57px;
  display: grid;
  min-width: 128px;
  padding: 8px 0;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 24px rgb(31 41 55 / 15%);
}
.organization-members-table__action-menu-panel button {
  padding: 9px 18px;
  border: 0;
  color: var(--el-text-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: 15px;
  text-align: left;
}
.organization-members-table__action-menu-panel button:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.organization-members-table__action-menu-panel--danger {
  color: var(--el-color-danger) !important;
}
.organization-members-table__footer {
  display: flex;
  min-height: 66px;
  padding: 0 30px;
  align-items: center;
  justify-content: space-between;
}
.organization-members-table__count {
  color: var(--el-text-color-primary);
  font-size: 15px;
}
.organization-members-table__page-size {
  width: 142px;
}

@media (max-width: 1160px) {
  .organization-members-table__toolbar {
    align-items: flex-start;
    padding: 18px 22px;
    flex-direction: column;
  }
  .organization-members-table__toolbar-right {
    width: 100%;
    flex-wrap: wrap;
  }
  .organization-members-table__search {
    flex: 1;
    min-width: 230px;
  }
}
</style>
