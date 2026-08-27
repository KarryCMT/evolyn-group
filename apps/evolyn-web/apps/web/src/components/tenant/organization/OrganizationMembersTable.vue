<script setup lang="ts">
import type {
  EvolynTableColumn,
  EvolynTableCustomRender,
  EvolynTableCustomRenderElement,
  EvolynTableCustomRenderObj,
} from '@evolyn.do/ui';
import { EvolynTable } from '@evolyn.do/ui';
import { RiArrowDownSFill, RiDownload2Line, RiSearch2Line } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import type { OrganizationMember, OrganizationMode } from './organization.types';

const props = defineProps<{
  mode: OrganizationMode;
  members: OrganizationMember[];
  keyword: string;
  status: 'all' | 'active' | 'disabled';
  total: number;
  currentPage: number;
  /** 租户创建人账号；其状态不可转为离职。 */
  tenantOwnerAccountId: string | null;
}>();

const emit = defineEmits<{
  'update:keyword': [value: string];
  'update:status': [value: 'all' | 'active' | 'disabled'];
  'update:page': [page: number];
  invite: [];
  addMember: [];
  export: [];
  import: [];
  edit: [member: OrganizationMember];
  handover: [member: OrganizationMember];
  disable: [member: OrganizationMember];
  remove: [member: OrganizationMember];
}>();

const ROW_HEIGHT = 64;
// VTable 的 customRender 坐标从单元格边缘起算，需显式预留与普通文本单元格一致的内边距。
const CELL_HORIZONTAL_PADDING = 12;
const actionMember = shallowRef<OrganizationMember | null>(null);
const actionMenuVisible = shallowRef(false);
const actionMemberIsTenantCreator = computed(
  () =>
    props.mode === 'department' &&
    props.tenantOwnerAccountId !== null &&
    actionMember.value?.accountId === props.tenantOwnerAccountId,
);

const tableRecords = computed(() =>
  props.members.map((member) => ({
    ...member,
    roleLabel: member.roleNames.join('、'),
  })),
);

function memberAt(row: number): OrganizationMember | undefined {
  // VTable 的第 0 行是表头；customRender 与 click-cell 返回的是表格行号，
  // 因此需换算为 records 下标。否则首条成员会回落为占位头像且姓名/角色为空。
  const recordIndex = row - 1;
  return recordIndex >= 0 ? props.members[recordIndex] : undefined;
}

function memberInitial(member: OrganizationMember | undefined) {
  return member?.name.trim().slice(0, 1) || '成';
}

const columns = computed<EvolynTableColumn[]>(() => {
  const avatar: EvolynTableCustomRenderElement = {
    type: 'circle',
    x: CELL_HORIZONTAL_PADDING + 14,
    y: ROW_HEIGHT / 2,
    radius: 14,
    fill: '#f25555',
  };
  const nameRender: EvolynTableCustomRender = ({ row }): EvolynTableCustomRenderObj => {
    const member = memberAt(row);
    return {
      expectedWidth: 270,
      expectedHeight: ROW_HEIGHT,
      elements: [
        avatar,
        {
          type: 'text',
          x: CELL_HORIZONTAL_PADDING + 14,
          y: ROW_HEIGHT / 2,
          text: memberInitial(member),
          fill: '#fff',
          fontSize: 14,
          textAlign: 'center',
          textBaseline: 'middle',
        },
        {
          type: 'text',
          x: CELL_HORIZONTAL_PADDING + 36,
          y: ROW_HEIGHT / 2,
          text: member?.name ?? '',
          fill: '#394150',
          fontSize: 16,
          textBaseline: 'middle',
        },
      ],
    };
  };
  const roleRender: EvolynTableCustomRender = ({ row }): EvolynTableCustomRenderObj => {
    const roleLabel = memberAt(row)?.roleNames.join('、') ?? '';
    return {
      expectedWidth: 190,
      expectedHeight: ROW_HEIGHT,
      elements: roleLabel
        ? [
            {
              type: 'rect',
              x: CELL_HORIZONTAL_PADDING,
              y: 17,
              width: 88,
              height: 30,
              fill: '#f1f5fd',
            },
            {
              type: 'text',
              x: CELL_HORIZONTAL_PADDING + 11,
              y: ROW_HEIGHT / 2,
              text: '●',
              fill: '#377ff5',
              fontSize: 13,
              textBaseline: 'middle',
            },
            {
              type: 'text',
              x: CELL_HORIZONTAL_PADDING + 27,
              y: ROW_HEIGHT / 2,
              text: roleLabel,
              fill: '#556071',
              fontSize: 15,
              textBaseline: 'middle',
            },
          ]
        : [],
    };
  };
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
      // 常见企业邮箱长度超过 170px，固定更宽列避免列表中被过早截断。
      { field: 'email', title: '邮箱', width: 240 },
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
  const member = memberAt(payload.row);
  if (member) openActionMenu(member);
}

function editMember() {
  if (actionMember.value) emit('edit', actionMember.value);
  actionMenuVisible.value = false;
}

function handoverCurrentMember() {
  if (actionMember.value) emit('handover', actionMember.value);
  actionMenuVisible.value = false;
}

function disableCurrentMember() {
  if (actionMember.value) emit('disable', actionMember.value);
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
            @update:model-value="emit('update:status', $event as 'all' | 'active' | 'disabled')"
          >
            <el-option label="全部" value="all" /><el-option
              label="已启用"
              value="active"
            /><el-option label="已停用" value="disabled" />
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
        <button v-if="props.mode === 'department'" type="button" @click="handoverCurrentMember">
          交接工作
        </button>
        <button v-if="props.mode === 'department'" type="button" @click="disableCurrentMember">
          停用
        </button>
        <button
          class="organization-members-table__action-menu-panel--danger"
          type="button"
          :disabled="actionMemberIsTenantCreator"
          :title="actionMemberIsTenantCreator ? '企业创建者无法转为离职' : undefined"
          @click="removeCurrentMember"
        >
          {{ props.mode === 'department' ? '转为离职' : '移出成员' }}
        </button>
      </div>
    </div>

    <footer class="organization-members-table__footer">
      <div class="organization-members-table__count">
        <el-select :model-value="20" class="organization-members-table__page-size" disabled
          ><el-option :value="20" label="20 条/页" /></el-select
        ><span>共 {{ props.total }} 条</span>
      </div>
      <el-pagination
        :current-page="props.currentPage"
        :page-size="20"
        :total="props.total"
        layout="prev, pager, next"
        @current-change="emit('update:page', $event)"
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
  padding: 0 var(--el-space-4xl);
  border-top: 1px solid var(--el-border-color-lighter);
  align-items: center;
  justify-content: space-between;
  gap: var(--el-space-2xl);
}
.organization-members-table__toolbar-left,
.organization-members-table__toolbar-right,
.organization-members-table__count {
  display: flex;
  align-items: center;
  gap: var(--el-space-lg);
}
.organization-members-table__toolbar-left :deep(.el-button) {
  height: 40px;
  font-size: var(--el-font-size-medium);
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
  padding: 0 var(--el-space-lg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-medium);
  align-items: center;
  gap: var(--el-space-md);
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
  font-size: var(--el-font-size-medium);
}
.organization-members-table__search input::placeholder {
  color: var(--el-text-color-placeholder);
}
.organization-members-table__status-label {
  color: var(--el-text-color-regular);
  font-size: var(--el-font-size-medium);
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
  padding: var(--el-space-md) 0;
  border-radius: var(--el-border-radius-medium);
  background: #fff;
  box-shadow: var(--el-box-shadow-light);
}
.organization-members-table__action-menu-panel button {
  padding: var(--el-space-md) var(--el-space-xl);
  border: 0;
  color: var(--el-text-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: var(--el-font-size-base);
  text-align: left;
}
.organization-members-table__action-menu-panel button:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.organization-members-table__action-menu-panel--danger {
  color: var(--el-color-danger) !important;
}
.organization-members-table__action-menu-panel button:disabled {
  color: var(--el-text-color-disabled) !important;
  background: transparent;
  cursor: not-allowed;
}
.organization-members-table__footer {
  display: flex;
  min-height: 66px;
  padding: 0 var(--el-space-4xl);
  align-items: center;
  justify-content: space-between;
}
.organization-members-table__count {
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-base);
}
.organization-members-table__page-size {
  width: 142px;
}

@media (max-width: 1160px) {
  .organization-members-table__toolbar {
    align-items: flex-start;
    padding: var(--el-space-xl) var(--el-space-2xl);
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
