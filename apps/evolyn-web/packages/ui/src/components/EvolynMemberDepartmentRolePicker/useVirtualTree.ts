import { computed, shallowRef, toValue, type MaybeRefOrGetter } from 'vue';
import type { EvolynMemberDepartmentRolePickerTreeNode } from './EvolynMemberDepartmentRolePicker.types';

/** 虚拟树渲染所需的扁平行，保留节点关系以支持缩进和展开操作。 */
export interface EvolynMemberDepartmentRolePickerTreeRow {
  depth: number;
  expanded: boolean;
  hasChildren: boolean;
  key: string;
  node: EvolynMemberDepartmentRolePickerTreeNode;
}

function createNodeKey(parentKey: string, node: EvolynMemberDepartmentRolePickerTreeNode) {
  return `${parentKey}/${String(node.id)}`;
}

/**
 * 将树按当前展开状态拍平，供虚拟列表只渲染视口中的行。
 * 默认全部展开，从而保持原组件的初始展示行为。
 */
export function useVirtualTree(
  nodes: MaybeRefOrGetter<EvolynMemberDepartmentRolePickerTreeNode[]>,
) {
  const collapsedKeys = shallowRef<ReadonlySet<string>>(new Set());

  const rows = computed<EvolynMemberDepartmentRolePickerTreeRow[]>(() => {
    const result: EvolynMemberDepartmentRolePickerTreeRow[] = [];

    function visit(
      currentNodes: EvolynMemberDepartmentRolePickerTreeNode[],
      depth: number,
      parentKey: string,
    ) {
      for (const node of currentNodes) {
        const key = createNodeKey(parentKey, node);
        const hasChildren = Boolean(node.children?.length);
        const expanded = hasChildren && !collapsedKeys.value.has(key);
        result.push({ depth, expanded, hasChildren, key, node });

        if (expanded) visit(node.children ?? [], depth + 1, key);
      }
    }

    visit(toValue(nodes), 0, 'root');
    return result;
  });

  function toggleExpanded(key: string) {
    const nextCollapsedKeys = new Set(collapsedKeys.value);
    if (nextCollapsedKeys.has(key)) {
      nextCollapsedKeys.delete(key);
    } else {
      nextCollapsedKeys.add(key);
    }
    collapsedKeys.value = nextCollapsedKeys;
  }

  return { rows, toggleExpanded };
}
