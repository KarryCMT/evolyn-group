import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import {
  createAdminGroup,
  getAdminGroup,
  listAdminGroups,
  updateAdminGroup,
  type AdminGroupPatchPayload,
  type AdminGroupSummaryDto,
} from '~/api/adminGroup';
import { ERROR_CODES } from '@evolyn.do/utils';
import type {
  AdministratorGroup,
  AdministratorScope,
} from '~/components/tenant/administrators/administrator.types';

/** 已知业务码 → 可操作文案；未知错误统一通用提示（禁止匹配错误文本）。 */
const errorMessages: Record<string, string> = {
  [ERROR_CODES.ADMIN_GROUP_DUPLICATE_NAME]: '管理组名称已存在',
  [ERROR_CODES.ADMIN_GROUP_NAME_INVALID]: '管理组名称不合法',
  [ERROR_CODES.ADMIN_GROUP_NOT_FOUND]: '管理组不存在，已刷新列表',
  [ERROR_CODES.ADMIN_GROUP_BUILTIN_IMMUTABLE]: '内置管理组不允许该操作',
  [ERROR_CODES.ADMIN_GROUP_LAST_ADMIN]: '系统管理员组至少保留一名管理员',
  [ERROR_CODES.ADMIN_GROUP_SCOPE_MISMATCH]: '该配置不适用于当前管理组类型',
  [ERROR_CODES.ADMIN_GROUP_CONFIG_INVALID]: '管理组配置不合法，已还原',
  [ERROR_CODES.ADMIN_GROUP_MEMBER_INVALID]: '成员不存在、跨租户或已离职',
  [ERROR_CODES.ADMIN_GROUP_TENANT_CREATOR_NOT_ALLOWED]: '企业创建者不能加入任何管理组',
};

function notifyError(err: unknown) {
  const code = (err as { errCode?: string } | null)?.errCode;
  ElMessage.error((code && errorMessages[code]) || '保存失败，请稍后重试');
}

/**
 * 管理员页状态编排（权限中心-管理员模块）：列表/详情读自 /admin-groups，
 * 面板每次勾选/选择确认即调一个配置区块的 PATCH（整体替换），响应为服务端
 * 最新详情——成功以响应覆盖本地状态，失败回滚乐观更新并提示。
 */
export function useAdministratorManagement(scope: AdministratorScope) {
  const groups = shallowRef<AdminGroupSummaryDto[]>([]);
  const selectedId = shallowRef<number | null>(null);
  const selectedGroup = shallowRef<AdministratorGroup | null>(null);
  const listLoading = shallowRef(false);
  const detailLoading = shallowRef(false);
  const saving = shallowRef(false);

  const builtInGroup = computed(() => groups.value.find((group) => group.builtIn));

  /** 加载列表并选中首个管理组（内置组恒在最前）。 */
  async function loadGroups(preferId?: number) {
    listLoading.value = true;
    try {
      groups.value = await listAdminGroups(scope);
      const next =
        (preferId && groups.value.find((group) => group.id === preferId)?.id) ||
        selectedId.value ||
        groups.value[0]?.id ||
        null;
      if (next !== selectedId.value) {
        await selectGroup(next);
      }
    } catch {
      ElMessage.error('管理组加载失败，请稍后重试');
    } finally {
      listLoading.value = false;
    }
  }

  /** 切换选中管理组并拉取详情。 */
  async function selectGroup(id: number | null) {
    selectedId.value = id;
    selectedGroup.value = null;
    if (!id) return;
    detailLoading.value = true;
    try {
      selectedGroup.value = await getAdminGroup(id);
    } catch {
      ElMessage.error('管理组详情加载失败');
    } finally {
      detailLoading.value = false;
    }
  }

  /** 新建管理组：成功后刷新列表并选中新组；失败返回 false（调用方保持弹窗）。 */
  async function addGroup(name: string): Promise<boolean> {
    try {
      const detail = await createAdminGroup({ scope, name });
      await loadGroups(detail.id);
      return true;
    } catch (err) {
      notifyError(err);
      return false;
    }
  }

  /**
   * 面板区块保存入口：本地字段 → 单区块 PATCH 载荷（后端至多接受一个区块，
   * 整体替换）。乐观更新本地状态，失败回滚快照；成功以服务端响应覆盖。
   */
  async function applyPatch(patch: Partial<AdministratorGroup>): Promise<boolean> {
    const group = selectedGroup.value;
    if (!group || saving.value) return false;

    const payload = buildPatchPayload(patch);
    if (!payload) return false;

    // 浅拷贝即回滚锚点：全部字段为一层平铺（数组/对象按引用整体还原）
    const snapshot = { ...group };
    saving.value = true;
    Object.assign(group, patch);
    try {
      selectedGroup.value = await updateAdminGroup(group.id, payload);
      return true;
    } catch (err) {
      Object.assign(group, snapshot);
      notifyError(err);
      if ((err as { errCode?: string } | null)?.errCode === ERROR_CODES.ADMIN_GROUP_NOT_FOUND) {
        void loadGroups();
      }
      return false;
    } finally {
      saving.value = false;
    }
  }

  return {
    groups,
    builtInGroup,
    selectedGroup,
    selectedId,
    listLoading,
    detailLoading,
    saving,
    loadGroups,
    selectGroup,
    addGroup,
    applyPatch,
  };
}

/**
 * 面板平铺字段 → 单区块 PATCH 载荷。调用约定：面板每次提交所属区块的
 * 全部字段（当前值 + 本次变更），此处不做跨对象回填；name/members 单独
 * 成块，范围类字段聚合为对应 scope 区块（mode=all 时后端会清空清单）。
 */
function buildPatchPayload(patch: Partial<AdministratorGroup>): AdminGroupPatchPayload | null {
  if (patch.name !== undefined) {
    return patch.name === '' ? null : { name: patch.name };
  }
  if (patch.members !== undefined) {
    return { members: patch.members.map((member) => member.id) };
  }
  if (
    patch.departmentEnabled !== undefined ||
    patch.departmentMode !== undefined ||
    patch.departmentIds !== undefined
  ) {
    return {
      departmentScope: {
        enabled: patch.departmentEnabled ?? false,
        mode: patch.departmentMode ?? 'partial',
        departmentIds: patch.departmentIds ?? [],
      },
    };
  }
  if (
    patch.roleVisible !== undefined ||
    patch.roleManage !== undefined ||
    patch.roleMode !== undefined ||
    patch.roleIds !== undefined
  ) {
    return {
      roleScope: {
        visible: patch.roleVisible ?? false,
        manage: patch.roleManage ?? false,
        mode: patch.roleMode ?? 'partial',
        roleIds: patch.roleIds ?? [],
      },
    };
  }
  if (patch.externalEnabled !== undefined) {
    return { externalOrg: { enabled: patch.externalEnabled } };
  }
  if (
    patch.applicationIds !== undefined ||
    patch.allApplications !== undefined ||
    patch.applicationManage !== undefined
  ) {
    return {
      applicationScope: {
        allApplications: patch.allApplications ?? false,
        applicationIds: patch.applicationIds ?? [],
        manage: patch.applicationManage ?? false,
      },
    };
  }
  if (patch.addressBook !== undefined) {
    return { addressBook: patch.addressBook };
  }
  return null;
}
