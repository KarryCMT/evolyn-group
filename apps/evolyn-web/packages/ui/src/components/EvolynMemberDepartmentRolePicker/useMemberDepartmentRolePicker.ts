import { computed, shallowRef, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue';
import type {
  EvolynMemberDepartmentRolePickerItemId,
  EvolynMemberDepartmentRolePickerItemType,
  EvolynMemberDepartmentRolePickerMember,
  EvolynMemberDepartmentRolePickerSelection,
  EvolynMemberDepartmentRolePickerTreeNode,
} from './EvolynMemberDepartmentRolePicker.types';

type PickerItem = EvolynMemberDepartmentRolePickerTreeNode | EvolynMemberDepartmentRolePickerMember;

interface UseMemberDepartmentRolePickerOptions {
  departments: MaybeRefOrGetter<EvolynMemberDepartmentRolePickerTreeNode[]>;
  roles: MaybeRefOrGetter<EvolynMemberDepartmentRolePickerTreeNode[]>;
  members: MaybeRefOrGetter<EvolynMemberDepartmentRolePickerMember[]>;
  selectableTypes: MaybeRefOrGetter<EvolynMemberDepartmentRolePickerItemType[]>;
  multiple: MaybeRefOrGetter<boolean>;
  departmentMultiple: MaybeRefOrGetter<boolean | undefined>;
  memberMultiple: MaybeRefOrGetter<boolean | undefined>;
  max: MaybeRefOrGetter<number | undefined>;
  selection: Ref<EvolynMemberDepartmentRolePickerSelection[]>;
}

function itemKey(
  type: EvolynMemberDepartmentRolePickerItemType,
  id: EvolynMemberDepartmentRolePickerItemId,
) {
  return `${type}:${String(id)}`;
}

function matchesKeywords(item: PickerItem, terms: string[]) {
  if (!terms.length) return true;

  const searchSource = [item.label, ...(item.keywords ?? [])].join(' ').toLocaleLowerCase();
  return terms.every((term) => searchSource.includes(term));
}

/** 搜索树时保留命中的祖先节点，避免结果失去组织层级。 */
function filterTreeNodes(
  nodes: EvolynMemberDepartmentRolePickerTreeNode[],
  terms: string[],
): EvolynMemberDepartmentRolePickerTreeNode[] {
  if (!terms.length) return nodes;

  return nodes.reduce<EvolynMemberDepartmentRolePickerTreeNode[]>((result, node) => {
    const children = filterTreeNodes(node.children ?? [], terms);
    const isMatched = matchesKeywords(node, terms);
    if (isMatched || children.length) {
      result.push({ ...node, children: isMatched ? node.children : children });
    }
    return result;
  }, []);
}

function findNode(
  nodes: EvolynMemberDepartmentRolePickerTreeNode[],
  id: EvolynMemberDepartmentRolePickerItemId,
): EvolynMemberDepartmentRolePickerTreeNode | undefined {
  for (const node of nodes) {
    if (String(node.id) === String(id)) return node;
    const found = findNode(node.children ?? [], id);
    if (found) return found;
  }
  return undefined;
}

function collectNodeIds(node: EvolynMemberDepartmentRolePickerTreeNode): Set<string> {
  const ids = new Set<string>([String(node.id)]);
  for (const child of node.children ?? []) {
    for (const id of collectNodeIds(child)) ids.add(id);
  }
  return ids;
}

function toSelection(
  item: PickerItem,
  type: EvolynMemberDepartmentRolePickerItemType,
): EvolynMemberDepartmentRolePickerSelection {
  return {
    id: item.id,
    label: item.label,
    type,
    ...('avatarUrl' in item && item.avatarUrl ? { avatarUrl: item.avatarUrl } : {}),
  };
}

/**
 * 管理弹窗内的草稿选择与检索派生数据。
 * 父组件传入的数据始终只读，确认前的所有变更均停留在 selection 草稿中。
 */
export function useMemberDepartmentRolePicker(options: UseMemberDepartmentRolePickerOptions) {
  const keyword = shallowRef('');
  const activeType = shallowRef<EvolynMemberDepartmentRolePickerItemType>('department');
  const activeDepartmentId = shallowRef<EvolynMemberDepartmentRolePickerItemId | undefined>();

  const searchTerms = computed(() =>
    keyword.value.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean),
  );
  const availableTypes = computed(() => toValue(options.selectableTypes));
  const selectedKeys = computed(
    () => new Set(options.selection.value.map((item) => itemKey(item.type, item.id))),
  );
  const visibleDepartments = computed(() =>
    filterTreeNodes(toValue(options.departments), searchTerms.value),
  );
  const visibleRoles = computed(() => filterTreeNodes(toValue(options.roles), searchTerms.value));
  const activeDepartmentIds = computed(() => {
    if (activeDepartmentId.value === undefined) return undefined;
    const activeDepartment = findNode(toValue(options.departments), activeDepartmentId.value);
    return activeDepartment ? collectNodeIds(activeDepartment) : undefined;
  });
  // 按部门预建成员索引，切换部门时不再扫描全部成员。
  const membersByDepartmentId = computed(() => {
    const index = new Map<string, EvolynMemberDepartmentRolePickerMember[]>();
    for (const member of toValue(options.members)) {
      for (const departmentId of member.departmentIds ?? []) {
        const key = String(departmentId);
        const bucket = index.get(key) ?? [];
        bucket.push(member);
        index.set(key, bucket);
      }
    }
    return index;
  });
  const visibleMembers = computed(() => {
    const departmentIds = activeDepartmentIds.value;
    if (!departmentIds) {
      return toValue(options.members).filter((member) =>
        matchesKeywords(member, searchTerms.value),
      );
    }

    // 成员可能属于多个部门，按成员 ID 去重后再执行关键词筛选。
    const scopedMembers = new Map<string, EvolynMemberDepartmentRolePickerMember>();
    for (const departmentId of departmentIds) {
      for (const member of membersByDepartmentId.value.get(departmentId) ?? []) {
        scopedMembers.set(String(member.id), member);
      }
    }
    return [...scopedMembers.values()].filter((member) =>
      matchesKeywords(member, searchTerms.value),
    );
  });

  watch(
    availableTypes,
    (types) => {
      if (!types.includes(activeType.value)) activeType.value = types[0] ?? 'department';
    },
    { immediate: true },
  );

  function isSelected(item: PickerItem, type: EvolynMemberDepartmentRolePickerItemType) {
    return selectedKeys.value.has(itemKey(type, item.id));
  }

  function isDisabled(item: PickerItem, type: EvolynMemberDepartmentRolePickerItemType) {
    if (item.disabled || ('selectable' in item && item.selectable === false)) return true;
    const isExisting = isSelected(item, type);
    const max = toValue(options.max);
    return !isExisting && max !== undefined && options.selection.value.length >= max;
  }

  /** 指定主体类型的单选配置优先，未配置时继续兼容原有的全局 multiple。 */
  function isMultiple(type: EvolynMemberDepartmentRolePickerItemType) {
    const departmentMultiple = toValue(options.departmentMultiple);
    if (type === 'department' && departmentMultiple !== undefined) {
      return departmentMultiple;
    }
    const memberMultiple = toValue(options.memberMultiple);
    if (type === 'member' && memberMultiple !== undefined) {
      return memberMultiple;
    }
    return toValue(options.multiple);
  }

  function hasTypeMultipleOverride(type: EvolynMemberDepartmentRolePickerItemType) {
    return (
      (type === 'department' && toValue(options.departmentMultiple) !== undefined) ||
      (type === 'member' && toValue(options.memberMultiple) !== undefined)
    );
  }

  function normalizeSelection(selections: EvolynMemberDepartmentRolePickerSelection[]) {
    return selections.reduce<EvolynMemberDepartmentRolePickerSelection[]>((result, selection) => {
      if (isMultiple(selection.type)) return [...result, selection];

      // 类型级单选只替换同类主体；全局单选保留原行为，替换整个选择结果。
      const retained = hasTypeMultipleOverride(selection.type)
        ? result.filter((item) => item.type !== selection.type)
        : [];
      return [...retained, selection];
    }, []);
  }

  function toggle(item: PickerItem, type: EvolynMemberDepartmentRolePickerItemType) {
    if (isDisabled(item, type)) return;

    const key = itemKey(type, item.id);
    if (selectedKeys.value.has(key)) {
      options.selection.value = options.selection.value.filter(
        (selection) => itemKey(selection.type, selection.id) !== key,
      );
      return;
    }

    const next = toSelection(item, type);
    if (isMultiple(type)) {
      options.selection.value = [...options.selection.value, next];
      return;
    }

    // 部门/成员独立单选不会干扰其他主体类型；未使用独立配置时维持全局单选语义。
    options.selection.value = hasTypeMultipleOverride(type)
      ? [...options.selection.value.filter((selection) => selection.type !== type), next]
      : [next];
  }

  function remove(selection: EvolynMemberDepartmentRolePickerSelection) {
    const source =
      selection.type === 'member'
        ? toValue(options.members).find((item) => String(item.id) === String(selection.id))
        : findNode(
            selection.type === 'department' ? toValue(options.departments) : toValue(options.roles),
            selection.id,
          );
    if (source?.disabled) return;

    const key = itemKey(selection.type, selection.id);
    options.selection.value = options.selection.value.filter(
      (item) => itemKey(item.type, item.id) !== key,
    );
  }

  function resetView() {
    keyword.value = '';
    activeDepartmentId.value = undefined;
    activeType.value = availableTypes.value[0] ?? 'department';
  }

  return {
    activeDepartmentId,
    activeType,
    availableTypes,
    isDisabled,
    isMultiple,
    isSelected,
    keyword,
    normalizeSelection,
    remove,
    resetView,
    selectedKeys,
    toggle,
    visibleDepartments,
    visibleMembers,
    visibleRoles,
  };
}
