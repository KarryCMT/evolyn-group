<script setup lang="ts">
import type { EvolynMemberDepartmentRolePickerSelection } from '@evolyn.do/ui';
import type { SaveFormPermissionGroupPayload } from '~/api/form';
import type {
  AssetPermissionGroup,
  PermissionFieldPermission,
  PermissionSubject,
} from '~/components/application/permissions/permission.types';
import { EvolynMemberDepartmentRolePicker } from '@evolyn.do/ui';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, shallowRef } from 'vue';
import { useRoute } from 'vue-router';
import {
  createFormPermissionGroup,
  deleteFormPermissionGroup,
  updateFormPermissionGroup,
} from '~/api/form';
import PermissionAssetList from '~/components/application/permissions/PermissionAssetList.vue';
import PermissionGroupEditorDialog from '~/components/application/permissions/PermissionGroupEditorDialog.vue';
import PermissionGroupsPanel from '~/components/application/permissions/PermissionGroupsPanel.vue';
import { useApplicationFormPermissions } from '~/composables/useApplicationFormPermissions';
import { useApplicationHome } from '~/composables/useApplicationHome';

defineOptions({ name: 'ApplicationSettingPermissionsPage' });

const route = useRoute();
const appCode = computed(() => String(route.params.appCode ?? ''));
const { application, errorMessage, reload, status } = useApplicationHome(appCode);
const permissionResources = useApplicationFormPermissions(appCode);

// 应用修改能力仅用于前端入口预检；最终授权始终由后端 form-permissions:* 校验。
const accessDenied = computed(
  () => status.value === 'ready' && !application.value?.capabilities.edit,
);
const keyword = shallowRef('');
const pickerVisible = shallowRef(false);
const targetGroupId = shallowRef<string>();
const pickerSelection = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);
const editorVisible = shallowRef(false);
const editingGroupId = shallowRef<string>();

const selectedGroups = computed(() => permissionResources.groups.value);
const pickerTitle = computed(() => (targetGroupId.value ? '添加授权对象' : '添加成员'));
const editingGroup = computed(() =>
  selectedGroups.value.find((group) => group.code === editingGroupId.value),
);

function openGroupPicker() {
  targetGroupId.value = undefined;
  pickerSelection.value = [];
  pickerVisible.value = true;
}

function openSubjectPicker(groupId: string) {
  targetGroupId.value = groupId;
  const group = selectedGroups.value.find((item) => item.code === groupId);
  pickerSelection.value = group?.subjects.map(toPickerSelection) ?? [];
  pickerVisible.value = true;
}

function toPickerSelection(subject: PermissionSubject): EvolynMemberDepartmentRolePickerSelection {
  return { id: subject.id, label: subject.name, type: subject.type };
}

function toPermissionSubject(
  selection: EvolynMemberDepartmentRolePickerSelection,
): PermissionSubject {
  return { id: Number(selection.id), name: selection.label, type: selection.type };
}

async function createOrUpdateSubjects(selections: EvolynMemberDepartmentRolePickerSelection[]) {
  const formCode = permissionResources.selectedFormCode.value;
  if (!formCode) return;
  const subjects = selections.map(toPermissionSubject);
  try {
    if (targetGroupId.value) {
      const group = selectedGroups.value.find((item) => item.code === targetGroupId.value);
      if (!group) return;
      await updateFormPermissionGroup(formCode, group.code, {
        ...payloadOf(group, subjects),
        baseRevision: group.revision,
      });
      ElMessage.success('授权对象已更新');
    } else {
      await createFormPermissionGroup(formCode, {
        ...defaultPayload(),
        subjectIds: subjects.map(toSubjectInput),
      });
      ElMessage.success('权限组已创建');
    }
    await permissionResources.refreshSelectedForm();
  } catch (error) {
    handleMutationError(error);
  }
}

function defaultPayload(): SaveFormPermissionGroupPayload {
  const fields = permissionResources.fields.value;
  const workflow = permissionResources.selectedAsset.value?.type === 'workflow-form';
  return {
    name: workflow ? '发起流程' : '管理全部数据',
    description: workflow ? '此分组内的成员可以发起流程。' : '此分组内的成员可以管理表单数据。',
    enabled: true,
    operations: workflow
      ? [
          'view',
          'add',
          'copy',
          'edit',
          'delete',
          'batch_print',
          'batch_modify',
          'import',
          'export',
          'workflow_owner_transfer',
          'workflow_terminate',
          'workflow_activate',
        ]
      : [
          'view',
          'add',
          'copy',
          'edit',
          'delete',
          'batch_print',
          'batch_modify',
          'import',
          'export',
        ],
    fieldPermissions: fields.map((field) => ({
      field: field.field,
      visible: true,
      editable: true,
    })),
    dataScope: { match: 'all', conditions: [] },
    subjectIds: [],
  };
}

/** 卡片编辑入口只保存当前组编码，弹窗提交后由页面统一写回服务端。 */
function openGroupEditor(groupId: string) {
  if (!selectedGroups.value.some((group) => group.code === groupId)) return;
  editingGroupId.value = groupId;
  editorVisible.value = true;
}

async function updatePermissionGroup(updatedGroup: AssetPermissionGroup) {
  const formCode = permissionResources.selectedFormCode.value;
  if (!formCode) return;
  try {
    await updateFormPermissionGroup(formCode, updatedGroup.code, {
      ...payloadOf(updatedGroup),
      baseRevision: updatedGroup.revision,
    });
    await permissionResources.refreshSelectedForm();
    ElMessage.success('权限组已更新');
  } catch (error) {
    handleMutationError(error);
  }
}

async function cloneGroup(groupId: string) {
  const formCode = permissionResources.selectedFormCode.value;
  const group = selectedGroups.value.find((item) => item.code === groupId);
  if (!formCode || !group) return;
  try {
    await createFormPermissionGroup(formCode, {
      ...payloadOf(group),
      name: `${group.name}（副本）`,
    });
    await permissionResources.refreshSelectedForm();
    ElMessage.success('权限组已复制');
  } catch (error) {
    handleMutationError(error);
  }
}

async function removeGroup(groupId: string) {
  const formCode = permissionResources.selectedFormCode.value;
  if (!formCode) return;
  try {
    await ElMessageBox.confirm('删除后该权限组中的成员将立即失去对应权限。', '删除权限组', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    });
    await deleteFormPermissionGroup(formCode, groupId);
    await permissionResources.refreshSelectedForm();
    ElMessage.success('已删除权限组');
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') handleMutationError(error);
  }
}

async function disableAll() {
  const formCode = permissionResources.selectedFormCode.value;
  if (!formCode || selectedGroups.value.length === 0) return;
  try {
    await ElMessageBox.confirm('停用后，当前资产下所有成员权限都会暂时失效。', '停用全部权限', {
      confirmButtonText: '确认停用',
      cancelButtonText: '取消',
      type: 'warning',
    });
    await Promise.all(
      selectedGroups.value
        .filter((group) => group.enabled)
        .map((group) =>
          updateFormPermissionGroup(formCode, group.code, {
            ...payloadOf(group),
            enabled: false,
            baseRevision: group.revision,
          }),
        ),
    );
    await permissionResources.refreshSelectedForm();
    ElMessage.success('已停用全部权限组');
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') handleMutationError(error);
  }
}

function updateGroupEnabled(payload: { groupId: string; enabled: boolean }) {
  const group = selectedGroups.value.find((item) => item.code === payload.groupId);
  if (!group) return;
  void updatePermissionGroup({ ...group, enabled: payload.enabled });
}

function payloadOf(
  group: AssetPermissionGroup,
  subjects = group.subjects,
): SaveFormPermissionGroupPayload {
  return {
    name: group.name,
    description: group.description,
    enabled: group.enabled,
    operations: group.operations,
    fieldPermissions: group.fields.map(toFieldRule),
    dataScope: group.dataScope,
    subjectIds: subjects.map(toSubjectInput),
  };
}

function toFieldRule(field: PermissionFieldPermission) {
  return { field: field.field, visible: field.visible, editable: field.editable };
}

function toSubjectInput(subject: PermissionSubject) {
  return { type: subject.type, id: subject.id };
}

function handleMutationError(error: unknown) {
  console.warn('[application-form-permissions] mutation failed', error);
  ElMessage.error('保存权限组失败；若配置已被他人修改，请刷新后重试。');
}
</script>

<template>
  <!-- 应用信息装载中：整页占位等待，避免准入判定闪烁。 -->
  <section
    v-if="status === 'loading'"
    v-loading="true"
    class="application-setting-permissions__status"
  />

  <el-result
    v-else-if="status === 'not-found'"
    class="application-setting-permissions__result"
    icon="warning"
    title="应用不存在或已不可访问"
    sub-title="请返回工作台后重新选择应用。"
  />

  <el-result
    v-else-if="status === 'error'"
    class="application-setting-permissions__result"
    icon="error"
    title="加载应用设置失败"
    :sub-title="errorMessage"
  >
    <template #extra>
      <el-button type="primary" @click="reload()"> 重新加载 </el-button>
    </template>
  </el-result>

  <!-- 仅应用管理员可进入权限设置，普通成员呈现无权限状态。 -->
  <el-result
    v-else-if="accessDenied"
    class="application-setting-permissions__result"
    icon="warning"
    title="无访问权限"
    sub-title="仅应用管理员可管理表单权限。"
  />

  <section
    v-else-if="permissionResources.status.value === 'loading'"
    v-loading="true"
    class="application-setting-permissions__status"
  />

  <el-result
    v-else-if="permissionResources.status.value === 'error'"
    class="application-setting-permissions__result"
    icon="error"
    title="加载表单权限失败"
    :sub-title="permissionResources.errorMessage.value"
  >
    <template #extra>
      <el-button type="primary" @click="permissionResources.reload()"> 重新加载 </el-button>
    </template>
  </el-result>

  <section v-else class="application-setting-permissions" aria-label="表单权限">
    <PermissionAssetList
      :assets="permissionResources.assets.value"
      :keyword="keyword"
      :selected-asset-id="permissionResources.selectedAssetId.value"
      @batch-select="ElMessage.info('批量配置尚未开放')"
      @update-keyword="keyword = $event"
      @select="permissionResources.selectedAssetId.value = $event"
    />
    <PermissionGroupsPanel
      :asset="permissionResources.selectedAsset.value"
      :groups="selectedGroups"
      @add-group="openGroupPicker"
      @add-subjects="openSubjectPicker"
      @clone-group="cloneGroup"
      @disable-all="disableAll"
      @edit-group="openGroupEditor"
      @remove-group="removeGroup"
      @update-group-enabled="updateGroupEnabled"
    />
    <PermissionGroupEditorDialog
      v-model="editorVisible"
      :asset-type="permissionResources.selectedAsset.value?.type"
      :group="editingGroup"
      :fields="permissionResources.fields.value"
      @confirm="updatePermissionGroup"
    />
    <EvolynMemberDepartmentRolePicker
      v-model="pickerSelection"
      v-model:open="pickerVisible"
      :departments="permissionResources.departments.value"
      :roles="permissionResources.roles.value"
      :members="permissionResources.members.value"
      :title="pickerTitle"
      @confirm="createOrUpdateSubjects"
    />
  </section>
</template>

<style scoped lang="scss">
.application-setting-permissions {
  display: flex;
  height: 100%;
  width: 100%;
  min-width: 920px;
  min-height: 0;
  overflow: hidden;

  &__status,
  &__result {
    display: grid;
    min-height: 100%;
    place-items: center;
  }
}

@media (max-width: 920px) {
  .application-setting-permissions {
    min-width: 760px;
  }
}
</style>
