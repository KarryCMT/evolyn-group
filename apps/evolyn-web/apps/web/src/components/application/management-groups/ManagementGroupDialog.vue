<script setup lang="ts">
import {
  EvolynMemberDepartmentRolePicker,
  type EvolynMemberDepartmentRolePickerItemType,
  type EvolynMemberDepartmentRolePickerMember,
  type EvolynMemberDepartmentRolePickerSelection,
  type EvolynMemberDepartmentRolePickerTreeNode,
} from '@evolyn.do/ui';
import {
  RiAddFill,
  RiCloseFill,
  RiDashboardFill,
  RiFileList3Fill,
  RiGitBranchFill,
} from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, reactive, shallowRef, watch } from 'vue';
import type {
  ManagementGroupDraft,
  ManagementGroupPermissionMode,
  ManagementGroupSubject,
} from './managementGroup.types';

defineOptions({ name: 'ManagementGroupDialog' });

type EditorStep = 'managers' | 'permissions' | 'scope';
type PickerTarget = 'managers' | 'scope';

interface ManagementAsset {
  id: string;
  name: string;
  type: 'form' | 'workflow' | 'dashboard';
}

const emit = defineEmits<{
  confirm: [draft: ManagementGroupDraft];
}>();

const visible = defineModel<boolean>({ default: false });
const activeStep = shallowRef<EditorStep>('managers');
const pickerTarget = shallowRef<PickerTarget>('managers');
const pickerVisible = shallowRef(false);
const pickerSelection = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);

const form = reactive({
  name: '',
  description: '',
  managers: [] as ManagementGroupSubject[],
  permissionMode: 'partial' as ManagementGroupPermissionMode,
  assetIds: [] as string[],
  departmentIds: ['department_company'] as string[],
  roleIds: ['role_test', 'role_sales_director', 'role_sales_manager', 'role_sales'] as string[],
});

/** 演示数据与权限页保持一致，接口到位后由页面注入真实组织和资产数据。 */
const assets: ManagementAsset[] = [
  { id: 'form_order', name: '订单管理', type: 'workflow' },
  { id: 'form_purchase', name: '采购申请', type: 'workflow' },
  { id: 'form_office', name: '办公用品申请', type: 'workflow' },
  { id: 'form_employee', name: '员工档案', type: 'form' },
  { id: 'form_product', name: '产品管理', type: 'form' },
  { id: 'form_customer', name: '客户信息', type: 'form' },
  { id: 'dashboard_employee', name: '员工信息分析', type: 'dashboard' },
  { id: 'dashboard_order', name: '订单分析', type: 'dashboard' },
  { id: 'dashboard_customer', name: '客户信息分析', type: 'dashboard' },
];

const departments: EvolynMemberDepartmentRolePickerTreeNode[] = [
  {
    id: 'department_company',
    label: '重庆万柯互联网科技有限责任公司',
    children: [
      { id: 'department_development', label: '研发部' },
      { id: 'department_product', label: '产品部' },
      { id: 'department_sales', label: '销售部' },
    ],
  },
];

const roles: EvolynMemberDepartmentRolePickerTreeNode[] = [
  { id: 'role_test', label: '测试' },
  { id: 'role_sales_director', label: '销售总监' },
  { id: 'role_sales_manager', label: '销售主管' },
  { id: 'role_sales', label: '销售人员' },
];

const members: EvolynMemberDepartmentRolePickerMember[] = [
  { id: 'member_lisi', label: '李同学', departmentIds: ['department_development'] },
  { id: 'member_zhangsan', label: '张三', departmentIds: ['department_product'] },
  { id: 'member_wangwu', label: '王五', departmentIds: ['department_sales'] },
];

const currentPickerTitle = computed(() =>
  pickerTarget.value === 'managers' ? '成员列表' : '选择管理范围',
);
const currentSelectableTypes = computed<EvolynMemberDepartmentRolePickerItemType[]>(() =>
  pickerTarget.value === 'managers' ? ['member'] : ['department', 'role'],
);
const currentPickerSelections = computed(() =>
  pickerTarget.value === 'managers'
    ? form.managers.map((manager) => ({
        id: manager.id,
        label: manager.name,
        type: 'member' as const,
      }))
    : [
        ...form.departmentIds.map((id) => toSelection(id, departments, 'department')),
        ...form.roleIds.map((id) => toSelection(id, roles, 'role')),
      ].filter((item): item is EvolynMemberDepartmentRolePickerSelection => Boolean(item)),
);

function findNode(
  id: string,
  nodes: EvolynMemberDepartmentRolePickerTreeNode[],
): EvolynMemberDepartmentRolePickerTreeNode | undefined {
  for (const node of nodes) {
    if (String(node.id) === id) return node;
    const child = node.children ? findNode(id, node.children) : undefined;
    if (child) return child;
  }
  return undefined;
}

function toSelection(
  id: string,
  nodes: EvolynMemberDepartmentRolePickerTreeNode[],
  type: 'department' | 'role',
): EvolynMemberDepartmentRolePickerSelection | undefined {
  const node = findNode(id, nodes);
  return node ? { id: node.id, label: node.label, type } : undefined;
}

function resetForm() {
  form.name = '';
  form.description = '';
  form.managers = [];
  form.permissionMode = 'partial';
  form.assetIds = [];
  form.departmentIds = ['department_company'];
  form.roleIds = ['role_test', 'role_sales_director', 'role_sales_manager', 'role_sales'];
  activeStep.value = 'managers';
}

function selectStep(step: EditorStep) {
  activeStep.value = step;
}

function openPicker(target: PickerTarget) {
  pickerTarget.value = target;
  pickerSelection.value = currentPickerSelections.value;
  pickerVisible.value = true;
}

function confirmPicker(selections: EvolynMemberDepartmentRolePickerSelection[]) {
  if (pickerTarget.value === 'managers') {
    form.managers = selections.map((selection) => ({
      id: String(selection.id),
      name: selection.label,
      type: 'member',
    }));
    return;
  }
  form.departmentIds = selections
    .filter((selection) => selection.type === 'department')
    .map((selection) => String(selection.id));
  form.roleIds = selections
    .filter((selection) => selection.type === 'role')
    .map((selection) => String(selection.id));
}

function removeManager(id: string) {
  form.managers = form.managers.filter((manager) => manager.id !== id);
}

function assetIcon(type: ManagementAsset['type']) {
  return type === 'dashboard'
    ? RiDashboardFill
    : type === 'workflow'
      ? RiGitBranchFill
      : RiFileList3Fill;
}

function submit() {
  if (!form.name.trim()) {
    ElMessage.warning('请输入管理组名称');
    return;
  }
  if (!form.managers.length) {
    activeStep.value = 'managers';
    ElMessage.warning('请至少添加一名管理员');
    return;
  }
  if (form.managers.some((manager) => manager.id === 'member_lisi')) {
    activeStep.value = 'managers';
    ElMessage.warning('企业创建者不能加入任何管理组');
    return;
  }
  if (form.permissionMode === 'partial' && !form.assetIds.length) {
    activeStep.value = 'permissions';
    ElMessage.warning('请选择可管理的表单或仪表盘');
    return;
  }
  emit('confirm', {
    name: form.name.trim(),
    description: form.description.trim(),
    managers: form.managers,
    permissionMode: form.permissionMode,
    assetIds: form.assetIds,
    departmentIds: form.departmentIds,
    roleIds: form.roleIds,
  });
  visible.value = false;
}

watch(visible, (isVisible) => {
  if (isVisible) resetForm();
});
</script>

<template>
  <el-dialog
    v-model="visible"
    class="management-group-dialog"
    width="min(960px, calc(100vw - 48px))"
    :show-close="false"
    :close-on-click-modal="false"
    append-to-body
    destroy-on-close
  >
    <template #header>
      <header class="management-group-dialog__header">
        <h2>添加管理组</h2>
        <button
          class="management-group-dialog__close"
          type="button"
          aria-label="关闭"
          @click="visible = false"
        >
          <RiCloseFill aria-hidden="true" />
        </button>
      </header>
    </template>

    <div class="management-group-dialog__body">
      <label class="management-group-dialog__field">
        <span>管理组名称<i>*</i></span>
        <el-input v-model="form.name" maxlength="30" placeholder="请输入内容" />
      </label>
      <label class="management-group-dialog__field">
        <span>描述信息</span>
        <el-input
          v-model="form.description"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 2 }"
          maxlength="200"
          placeholder="填写描述信息，让其他管理员了解此管理组的用途和注意事项"
        />
      </label>

      <section class="management-group-dialog__editor" aria-label="管理组配置">
        <nav class="management-group-dialog__steps" aria-label="配置步骤">
          <button
            v-for="step in [
              { value: 'managers', label: '添加管理员' },
              { value: 'permissions', label: '选择权限' },
              { value: 'scope', label: '配置管理范围' },
            ]"
            :key="step.value"
            class="management-group-dialog__step"
            :class="{ 'management-group-dialog__step--active': activeStep === step.value }"
            type="button"
            @click="selectStep(step.value as EditorStep)"
          >
            {{ step.label }}
          </button>
        </nav>

        <div class="management-group-dialog__panel">
          <template v-if="activeStep === 'managers'">
            <h3>管理员</h3>
            <button
              class="management-group-dialog__selection-box"
              type="button"
              aria-label="选择管理员"
              @click="openPicker('managers')"
            >
              <template v-if="form.managers.length">
                <span
                  v-for="manager in form.managers"
                  :key="manager.id"
                  class="management-group-dialog__member-tag"
                >
                  <i>{{ manager.name.slice(0, 1) }}</i>
                  {{ manager.name }}
                  <RiCloseFill aria-hidden="true" @click.stop="removeManager(manager.id)" />
                </span>
              </template>
              <span v-else class="management-group-dialog__select-hint"
                ><RiAddFill aria-hidden="true" />选择成员</span
              >
            </button>
          </template>

          <template v-else-if="activeStep === 'permissions'">
            <h3>权限范围</h3>
            <el-select
              v-model="form.permissionMode"
              class="management-group-dialog__permission-select"
            >
              <el-option label="仅拥有部分表单/仪表盘权限" value="partial" />
              <el-option label="拥有应用全部权限" value="all" />
            </el-select>
            <div
              v-if="form.permissionMode === 'partial'"
              class="management-group-dialog__asset-picker"
            >
              <el-checkbox-group
                v-model="form.assetIds"
                class="management-group-dialog__asset-options"
              >
                <el-checkbox v-for="asset in assets" :key="asset.id" :value="asset.id">
                  <component :is="assetIcon(asset.type)" aria-hidden="true" />
                  {{ asset.name }}
                </el-checkbox>
              </el-checkbox-group>
              <div class="management-group-dialog__selected-assets" aria-label="已选择权限">
                <span v-if="!form.assetIds.length" class="management-group-dialog__empty-selection"
                  >请选择表单或仪表盘</span
                >
                <span
                  v-for="asset in assets.filter((item) => form.assetIds.includes(item.id))"
                  :key="asset.id"
                  class="management-group-dialog__asset-tag"
                >
                  <component :is="assetIcon(asset.type)" aria-hidden="true" />
                  {{ asset.name }}
                </span>
              </div>
            </div>
          </template>

          <template v-else>
            <div class="management-group-dialog__scope-heading">
              <h3>管理范围</h3>
              <p>控制表单&amp;流程&amp;仪表盘设计和发布中，部门和角色的选择范围。</p>
              <button type="button" @click="ElMessage.info('帮助文档将在后续版本开放')">
                帮助文档
              </button>
            </div>
            <div class="management-group-dialog__scope-section">
              <h4>部门管理范围</h4>
              <button
                class="management-group-dialog__selection-box"
                type="button"
                @click="openPicker('scope')"
              >
                <span
                  v-for="selection in currentPickerSelections.filter(
                    (item) => item.type === 'department',
                  )"
                  :key="selection.id"
                  class="management-group-dialog__scope-tag"
                >
                  {{ selection.label }}
                </span>
                <span v-if="!form.departmentIds.length" class="management-group-dialog__select-hint"
                  ><RiAddFill aria-hidden="true" />选择部门</span
                >
              </button>
            </div>
            <div class="management-group-dialog__scope-section">
              <h4>角色管理范围</h4>
              <button
                class="management-group-dialog__selection-box"
                type="button"
                @click="openPicker('scope')"
              >
                <span
                  v-for="selection in currentPickerSelections.filter(
                    (item) => item.type === 'role',
                  )"
                  :key="selection.id"
                  class="management-group-dialog__scope-tag management-group-dialog__scope-tag--role"
                >
                  {{ selection.label }}
                </span>
                <span v-if="!form.roleIds.length" class="management-group-dialog__select-hint"
                  ><RiAddFill aria-hidden="true" />选择角色</span
                >
              </button>
            </div>
          </template>
        </div>
      </section>
    </div>

    <template #footer>
      <footer class="management-group-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="submit">确定</el-button>
      </footer>
    </template>
  </el-dialog>

  <EvolynMemberDepartmentRolePicker
    v-model="pickerSelection"
    v-model:open="pickerVisible"
    :title="currentPickerTitle"
    :departments="departments"
    :roles="roles"
    :members="members"
    :selectable-types="currentSelectableTypes"
    :allow-empty="pickerTarget === 'scope'"
    @confirm="confirmPicker"
  />
</template>

<!-- 弹窗传送到 body，样式必须以唯一块类收敛，避免影响其他设置页的弹窗。 -->
<style lang="scss">
.management-group-dialog.el-dialog {
  display: flex;
  max-width: calc(100vw - 32px);
  height: min(860px, calc(100vh - 44px));
  margin: var(--el-space-2xl) auto;
  overflow: hidden;
  flex-direction: column;
  border-radius: var(--el-border-radius-large);
}

.management-group-dialog .el-dialog__header,
.management-group-dialog .el-dialog__footer {
  flex: 0 0 auto;
  padding: 0;
  margin: 0;
}

.management-group-dialog .el-dialog__header {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.management-group-dialog .el-dialog__body {
  display: flex;
  min-height: 0;
  padding: 0;
  flex: 1;
  overflow: hidden;
}

.management-group-dialog .el-dialog__footer {
  border-top: 1px solid var(--el-border-color-lighter);
}

.management-group-dialog__header {
  display: flex;
  height: 56px;
  padding: 0 var(--el-space-xl) 0 var(--el-space-3xl);
  align-items: center;
  justify-content: space-between;

  h2 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-extra-large);
    font-weight: 650;
    line-height: 26px;
  }
}

.management-group-dialog__close {
  display: inline-flex;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 0;
  border-radius: var(--el-border-radius-base);
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;

  svg {
    width: 22px;
    height: 22px;
  }

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}

.management-group-dialog__body {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  flex: 1;
  padding: var(--el-space-3xl) var(--el-space-3xl);
  flex-direction: column;
  gap: var(--el-space-xl);
}

.management-group-dialog__field {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: var(--el-space-md);
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-medium);
  line-height: 24px;

  span i {
    margin-left: var(--el-space-xs);
    color: var(--el-color-danger);
    font-style: normal;
  }

  .el-input__wrapper,
  .el-textarea__inner {
    box-shadow: 0 0 0 1px var(--el-border-color) inset;
  }

  .el-input__wrapper {
    min-height: 42px;
  }

  .el-textarea__inner {
    padding: var(--el-space-md) var(--el-space-lg);
    resize: none;
  }
}

.management-group-dialog__editor {
  display: flex;
  min-height: 0;
  flex: 1;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-large);
  overflow: hidden;
}

.management-group-dialog__steps {
  display: flex;
  width: 190px;
  flex: 0 0 190px;
  flex-direction: column;
  border-right: 1px solid var(--el-border-color);
}

.management-group-dialog__step {
  min-height: 56px;
  padding: 0 var(--el-space-3xl);
  border: 0;
  border-left: 4px solid var(--el-color-transparent);
  color: var(--el-text-color-primary);
  cursor: pointer;
  background: transparent;
  font: inherit;
  font-size: var(--el-font-size-medium);
  text-align: left;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &--active {
    border-left-color: var(--el-color-primary);
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    font-weight: 600;
  }
}

.management-group-dialog__panel {
  min-width: 0;
  padding: var(--el-space-3xl);
  flex: 1;

  h3,
  h4,
  p {
    margin: 0;
  }

  h3 {
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    line-height: 26px;
  }
}

.management-group-dialog__selection-box {
  display: flex;
  width: 100%;
  min-height: 112px;
  margin-top: var(--el-space-lg);
  padding: var(--el-space-lg);
  border: 1px dashed var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  align-content: flex-start;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: var(--el-space-md);
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: var(--el-fill-color-blank);
  font: inherit;
  text-align: left;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    background: var(--el-color-primary-light-9);
  }
}

.management-group-dialog__select-hint {
  display: inline-flex;
  width: 100%;
  min-height: 82px;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  gap: var(--el-space-sm);
}

.management-group-dialog__member-tag,
.management-group-dialog__scope-tag,
.management-group-dialog__asset-tag {
  display: inline-flex;
  min-height: 34px;
  padding: 0 var(--el-space-md);
  border-radius: var(--el-border-radius-small);
  align-items: center;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  font-size: var(--el-font-size-base);
  gap: var(--el-space-sm);
}

.management-group-dialog__member-tag {
  i {
    display: grid;
    width: 22px;
    height: 22px;
    border-radius: var(--el-border-radius-half);
    place-items: center;
    color: var(--el-color-white);
    background: var(--el-color-danger);
    font-size: var(--el-font-size-extra-small);
    font-style: normal;
  }

  svg {
    width: 16px;
    height: 16px;
    color: var(--el-text-color-secondary);
    cursor: pointer;

    &:hover {
      color: var(--el-color-danger);
    }
  }
}

.management-group-dialog__permission-select {
  width: 100%;
  margin-top: var(--el-space-lg);
}

.management-group-dialog__asset-picker {
  display: grid;
  min-height: 260px;
  margin-top: var(--el-space-lg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
}

.management-group-dialog__asset-options {
  display: flex;
  min-height: 0;
  padding: var(--el-space-xl);
  flex-direction: column;
  gap: var(--el-space-lg);
  border-right: 1px solid var(--el-border-color);

  .el-checkbox {
    margin-right: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-base);
  }

  .el-checkbox__label {
    display: inline-flex;
    align-items: center;
    gap: var(--el-space-sm);
  }

  svg {
    width: 17px;
    height: 17px;
    color: var(--el-color-primary);
  }
}

.management-group-dialog__selected-assets {
  display: flex;
  padding: var(--el-space-xl);
  align-content: flex-start;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: var(--el-space-md);
}

.management-group-dialog__asset-tag {
  svg {
    width: 16px;
    height: 16px;
    color: var(--el-color-primary);
  }
}

.management-group-dialog__empty-selection {
  color: var(--el-text-color-placeholder);
  font-size: var(--el-font-size-base);
}

.management-group-dialog__scope-heading {
  display: flex;
  align-items: center;
  gap: var(--el-space-lg);

  p {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 22px;
  }

  button {
    padding: var(--el-space-xs);
    border: 0;
    border-radius: var(--el-border-radius-small);
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    font: inherit;
    font-size: var(--el-font-size-base);

    &:hover {
      background: var(--el-color-primary-light-9);
    }
  }
}

.management-group-dialog__scope-section {
  margin-top: var(--el-space-3xl);

  h4 {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    font-weight: 500;
    line-height: 22px;
  }
}

.management-group-dialog__scope-tag {
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);

  &--role {
    color: var(--el-color-primary);
  }
}

.management-group-dialog__footer {
  display: flex;
  height: 66px;
  padding: 0 var(--el-space-3xl);
  align-items: center;
  justify-content: flex-end;
  gap: var(--el-space-md);
}

@media (max-width: 720px) {
  .management-group-dialog.el-dialog {
    height: calc(100vh - 20px);
    margin: var(--el-space-md) auto;
  }

  .management-group-dialog__body {
    padding: var(--el-space-xl);
  }

  .management-group-dialog__editor {
    flex-direction: column;
  }

  .management-group-dialog__steps {
    width: 100%;
    flex: 0 0 auto;
    flex-direction: row;
    border-right: 0;
    border-bottom: 1px solid var(--el-border-color);
  }

  .management-group-dialog__step {
    min-height: 46px;
    padding: 0 var(--el-space-lg);
    border-bottom: 3px solid var(--el-color-transparent);
    border-left: 0;
    font-size: var(--el-font-size-base);

    &--active {
      border-bottom-color: var(--el-color-primary);
    }
  }

  .management-group-dialog__panel {
    padding: var(--el-space-xl);
  }

  .management-group-dialog__asset-picker {
    grid-template-columns: 1fr;
  }

  .management-group-dialog__asset-options {
    border-right: 0;
    border-bottom: 1px solid var(--el-border-color);
  }

  .management-group-dialog__scope-heading {
    align-items: flex-start;
    flex-wrap: wrap;
  }
}
</style>
