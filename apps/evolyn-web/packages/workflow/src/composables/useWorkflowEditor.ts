import { computed, shallowRef, type ComputedRef, type ShallowRef } from 'vue';
import { cloneWorkflowDocument, type WorkflowDocument, type WorkflowNode } from '../schema';

export interface WorkflowEditor {
  document: ShallowRef<WorkflowDocument>;
  selectedNodeId: ShallowRef<string | null>;
  selectedNode: ComputedRef<WorkflowNode | null>;
  selectNode: (nodeId: string | null) => void;
  updateNode: (nodeId: string, patch: Partial<WorkflowNode>) => void;
  replaceDocument: (document: WorkflowDocument) => void;
}

/** 编辑器状态只服务于 Schema，不包含 LogicFlow 实例或页面级保存行为。 */
export function useWorkflowEditor(initialDocument: WorkflowDocument): WorkflowEditor {
  const document = shallowRef(cloneWorkflowDocument(initialDocument));
  const selectedNodeId = shallowRef<string | null>(document.value.nodes[0]?.id ?? null);
  const selectedNode = computed(
    () => document.value.nodes.find((node) => node.id === selectedNodeId.value) ?? null,
  );

  function selectNode(nodeId: string | null) {
    selectedNodeId.value = document.value.nodes.some((node) => node.id === nodeId) ? nodeId : null;
  }

  function updateNode(nodeId: string, patch: Partial<WorkflowNode>) {
    document.value = {
      ...document.value,
      nodes: document.value.nodes.map((node) =>
        node.id === nodeId
          ? {
              ...node,
              ...patch,
              ...(patch.position ? { position: { ...patch.position } } : {}),
              ...(patch.fieldPermissions
                ? { fieldPermissions: cloneFieldPermissions(patch.fieldPermissions) }
                : {}),
              ...(patch.config ? { config: { ...patch.config } } : {}),
            }
          : node,
      ),
    };
  }

  function replaceDocument(value: WorkflowDocument) {
    document.value = cloneWorkflowDocument(value);
    selectNode(document.value.nodes[0]?.id ?? null);
  }

  return { document, selectedNodeId, selectedNode, selectNode, updateNode, replaceDocument };
}

function cloneFieldPermissions(permissions: WorkflowNode['fieldPermissions']) {
  return Object.fromEntries(
    Object.entries(permissions).map(([fieldId, permission]) => [fieldId, { ...permission }]),
  );
}
