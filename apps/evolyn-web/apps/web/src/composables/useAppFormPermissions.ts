import type {
  EvolynMemberDepartmentRolePickerMember,
  EvolynMemberDepartmentRolePickerTreeNode,
} from '@evolyn.do/ui';
import type { Ref } from 'vue';
import type {
  AssetPermissionGroup,
  PermissionAsset,
  PermissionField,
} from '~/components/app/permissions/permission.types';
import { computed, readonly, shallowRef, watch } from 'vue';
import { getAppMenuByCode } from '~/api/apps';
import { getDepartmentTree } from '~/api/department';
import { listFormPermissionFields, listFormPermissionGroups } from '~/api/form';
import { listMembers } from '~/api/member';
import { getOrganizationRoleTree } from '~/api/role';

export type FormPermissionResourceStatus = 'loading' | 'ready' | 'error';
export type FormPermissionDetailStatus = 'idle' | 'loading' | 'ready' | 'error';

/**
 * 应用权限页的数据编排：菜单提供表单树，IAM 提供可授权主体，表单权限接口
 * 提供当前选中表单的字段与权限组。页面仅消费状态并向此 composable 请求刷新。
 */
export function useAppFormPermissions(appCode: Readonly<Ref<string>>) {
  const assets = shallowRef<PermissionAsset[]>([]);
  const selectedAssetId = shallowRef('');
  const groups = shallowRef<AssetPermissionGroup[]>([]);
  const fields = shallowRef<PermissionField[]>([]);
  const departments = shallowRef<EvolynMemberDepartmentRolePickerTreeNode[]>([]);
  const roles = shallowRef<EvolynMemberDepartmentRolePickerTreeNode[]>([]);
  const members = shallowRef<EvolynMemberDepartmentRolePickerMember[]>([]);
  const status = shallowRef<FormPermissionResourceStatus>('loading');
  // 资产树和当前表单权限详情分别加载；切换左栏资产时不能让整页回到 loading，
  // 否则资产树会被卸载并重建，导致滚动位置和展开状态丢失。
  const selectedFormStatus = shallowRef<FormPermissionDetailStatus>('idle');
  const errorMessage = shallowRef('');
  const selectedFormErrorMessage = shallowRef('');
  let resourceRequest = 0;
  let formRequest = 0;

  const selectedAsset = computed(() => findAssetByID(assets.value, selectedAssetId.value));
  const selectedFormCode = computed(() => selectedAsset.value?.formCode ?? '');

  async function loadResources(code = appCode.value) {
    const request = ++resourceRequest;
    if (!code) {
      assets.value = [];
      selectedAssetId.value = '';
      selectedFormStatus.value = 'idle';
      selectedFormErrorMessage.value = '';
      status.value = 'ready';
      return;
    }
    status.value = 'loading';
    errorMessage.value = '';
    try {
      const [menu, departmentTree, roleTree, memberItems] = await Promise.all([
        getAppMenuByCode(code),
        getDepartmentTree(),
        getOrganizationRoleTree(),
        loadAllActiveMembers(),
      ]);
      if (request !== resourceRequest) return;
      assets.value = formAssetsFromMenu(menu);
      departments.value = departmentTree.map(toDepartmentNode);
      roles.value = roleTree.groups.map((group) => ({
        id: group.id,
        label: group.name,
        selectable: false,
        children: group.roles.map((role) => ({ id: role.id, label: role.name })),
      }));
      members.value = memberItems.map((member) => ({
        id: member.id,
        label: member.name,
        avatarUrl: member.avatar || undefined,
        departmentIds: member.departments.map((department) => department.id),
        keywords: [member.phone, member.email].filter(Boolean),
      }));
      const firstForm = firstFormAsset(assets.value);
      selectedAssetId.value = findAssetByID(assets.value, selectedAssetId.value)
        ? selectedAssetId.value
        : (firstForm?.id ?? '');
      status.value = 'ready';
    } catch (error) {
      if (request !== resourceRequest) return;
      console.warn('[app-form-permissions] load resources failed', error);
      assets.value = [];
      selectedAssetId.value = '';
      selectedFormStatus.value = 'idle';
      selectedFormErrorMessage.value = '';
      status.value = 'error';
      errorMessage.value = '表单权限配置资源加载失败，请稍后重试。';
    }
  }

  async function refreshSelectedForm() {
    const formCode = selectedFormCode.value;
    const request = ++formRequest;
    groups.value = [];
    fields.value = [];
    if (!formCode) {
      selectedFormStatus.value = 'idle';
      selectedFormErrorMessage.value = '';
      return;
    }
    selectedFormStatus.value = 'loading';
    selectedFormErrorMessage.value = '';
    try {
      const [nextGroups, nextFields] = await Promise.all([
        listFormPermissionGroups(formCode),
        listFormPermissionFields(formCode),
      ]);
      if (request !== formRequest) return;
      fields.value = nextFields;
      groups.value = nextGroups.map((group) => ({
        ...group,
        fields: permissionFieldsFrom(group.fieldPermissions, nextFields),
      }));
      selectedFormStatus.value = 'ready';
    } catch (error) {
      if (request !== formRequest) return;
      console.warn('[app-form-permissions] load permission groups failed', error);
      selectedFormStatus.value = 'error';
      selectedFormErrorMessage.value = '表单权限组加载失败，请刷新后重试。';
    }
  }

  watch(appCode, () => void loadResources(), { immediate: true });
  watch(selectedFormCode, () => void refreshSelectedForm());

  return {
    assets,
    selectedAssetId,
    selectedAsset,
    selectedFormCode: readonly(selectedFormCode),
    groups,
    fields,
    departments,
    roles,
    members,
    status: readonly(status),
    errorMessage: readonly(errorMessage),
    selectedFormStatus: readonly(selectedFormStatus),
    selectedFormErrorMessage: readonly(selectedFormErrorMessage),
    reload: loadResources,
    refreshSelectedForm,
  };
}

async function loadAllActiveMembers() {
  const pageSize = 100;
  const result = [] as Awaited<ReturnType<typeof listMembers>>['items'];
  for (let page = 1; ; page += 1) {
    const response = await listMembers({ status: 'active', page, pageSize });
    result.push(...response.items);
    if (result.length >= response.total || response.items.length < pageSize) return result;
  }
}

function formAssetsFromMenu(menu: Awaited<ReturnType<typeof getAppMenuByCode>>): PermissionAsset[] {
  const childrenByParent = new Map<string | null, (typeof menu.nodeMap)[string][]>();
  for (const node of Object.values(menu.nodeMap)) {
    const children = childrenByParent.get(node.parentMenuId) ?? [];
    children.push(node);
    childrenByParent.set(node.parentMenuId, children);
  }
  const build = (parentID: string | null): PermissionAsset[] =>
    (childrenByParent.get(parentID) ?? [])
      .sort(
        (left, right) =>
          left.sortOrder - right.sortOrder || left.menuId.localeCompare(right.menuId),
      )
      .flatMap((node): PermissionAsset[] => {
        if (node.type === 'group') {
          const children = build(node.menuId);
          return children.length
            ? [{ id: node.menuId, name: node.name, type: 'group', children }]
            : [];
        }
        if (node.type !== 'form' || node.target?.type !== 'form') return [];
        return [
          {
            id: node.menuId,
            name: node.name,
            type: node.target.formType === 'workflow' ? 'workflow-form' : 'form',
            formCode: node.target.code,
          },
        ];
      });
  return build(null);
}

function toDepartmentNode(
  department: Awaited<ReturnType<typeof getDepartmentTree>>[number],
): EvolynMemberDepartmentRolePickerTreeNode {
  return {
    id: department.id,
    label: department.name,
    children: department.children?.map(toDepartmentNode),
  };
}

function findAssetByID(list: readonly PermissionAsset[], id: string): PermissionAsset | undefined {
  for (const asset of list) {
    if (asset.id === id) return asset;
    const found = asset.children ? findAssetByID(asset.children, id) : undefined;
    if (found) return found;
  }
  return undefined;
}

function firstFormAsset(list: readonly PermissionAsset[]): PermissionAsset | undefined {
  for (const asset of list) {
    if (asset.formCode) return asset;
    const found = asset.children ? firstFormAsset(asset.children) : undefined;
    if (found) return found;
  }
  return undefined;
}

function permissionFieldsFrom(
  rules: readonly { field: string; visible: boolean; editable: boolean }[],
  fields: readonly PermissionField[],
) {
  const rulesByField = new Map(rules.map((rule) => [rule.field, rule]));
  return fields.map((field) => ({
    field: field.field,
    label: field.label,
    required: field.required,
    visible: rulesByField.get(field.field)?.visible ?? false,
    editable: rulesByField.get(field.field)?.editable ?? false,
  }));
}
