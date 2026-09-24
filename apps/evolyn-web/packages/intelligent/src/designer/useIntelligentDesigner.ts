import { computed, shallowRef } from 'vue';
import {
  type IntelligentActionType,
  type IntelligentDocument,
  type IntelligentNode,
  type IntelligentPosition,
  type IntelligentTrigger,
  addActionNode,
  moveIntelligentNode,
  removeActionNode,
  updateIntelligentNode,
  updateIntelligentTrigger,
} from '../schema';
import { IntelligentContext } from '../context/IntelligentContext';
import { intelligentNodeTemplates } from '../mock/nodeTemplates';
import { layoutIntelligentDocument } from '../util';

interface UseIntelligentDesignerOptions {
  getDocument: () => IntelligentDocument;
  updateDocument: (document: IntelligentDocument) => void;
}

/** 管理设计器选择、编辑与历史，UI 组件只负责显式派发用户意图。 */
export function useIntelligentDesigner(options: UseIntelligentDesignerOptions) {
  const selectedNodeId = shallowRef<string | null>(null);
  const canUndo = shallowRef(false);
  const canRedo = shallowRef(false);
  const context = new IntelligentContext(options.getDocument());

  const selectedAction = computed(() =>
    options
      .getDocument()
      .nodes.find(
        (node) => node.id === selectedNodeId.value && node.type === 'action',
      ),
  );

  function commit(document: IntelligentDocument): void {
    const snapshot = context.history.add(document);
    canUndo.value = snapshot.canUndo;
    canRedo.value = snapshot.canRedo;
    options.updateDocument(document);
    context.events.emit('document:change', document);
  }

  function addNode(actionType: IntelligentActionType = 'create-record'): void {
    const template = intelligentNodeTemplates.find((item) => item.type === actionType);
    const next = addActionNode(options.getDocument(), {
      actionType,
      name: template?.name,
      description: template?.description,
    });
    if (next === options.getDocument()) return;
    commit(next);
    const actions = next.nodes.filter((node) => node.type === 'action');
    selectedNodeId.value = actions[actions.length - 1]?.id ?? null;
  }

  function deleteSelected(): void {
    if (selectedAction.value) removeNode(selectedAction.value.id);
  }

  function removeNode(nodeId: string): void {
    const next = removeActionNode(options.getDocument(), nodeId);
    if (next === options.getDocument()) return;
    commit(next);
    if (selectedNodeId.value === nodeId) selectedNodeId.value = null;
  }

  function moveNode(nodeId: string, position: IntelligentPosition): void {
    commit(moveIntelligentNode(options.getDocument(), nodeId, position));
  }

  function updateNode(
    nodeId: string,
    patch: Partial<Omit<IntelligentNode, 'id' | 'type'>>,
  ): void {
    commit(updateIntelligentNode(options.getDocument(), nodeId, patch));
  }

  function updateTrigger(patch: Partial<IntelligentTrigger>): void {
    commit(updateIntelligentTrigger(options.getDocument(), patch));
  }

  function beautify(): void {
    commit(layoutIntelligentDocument(options.getDocument()));
  }

  function undo(): void {
    const snapshot = context.history.undo();
    if (!snapshot.current) return;
    canUndo.value = snapshot.canUndo;
    canRedo.value = snapshot.canRedo;
    options.updateDocument(snapshot.current);
  }

  function redo(): void {
    const snapshot = context.history.redo();
    if (!snapshot.current) return;
    canUndo.value = snapshot.canUndo;
    canRedo.value = snapshot.canRedo;
    options.updateDocument(snapshot.current);
  }

  function resetHistory(document: IntelligentDocument): void {
    context.history.reset(document);
    canUndo.value = false;
    canRedo.value = false;
    selectedNodeId.value = null;
  }

  function handleKeyboard(event: KeyboardEvent): void {
    const command = event.metaKey || event.ctrlKey;
    if (command && event.key.toLowerCase() === 'z') {
      event.preventDefault();
      event.shiftKey ? redo() : undo();
      return;
    }
    if (command && event.key.toLowerCase() === 'y') {
      event.preventDefault();
      redo();
      return;
    }
    if (event.key === 'Delete' || event.key === 'Backspace') {
      event.preventDefault();
      deleteSelected();
    }
  }

  return {
    addNode,
    beautify,
    canRedo,
    canUndo,
    deleteSelected,
    handleKeyboard,
    moveNode,
    redo,
    removeNode,
    resetHistory,
    selectedAction,
    selectedNodeId,
    undo,
    updateNode,
    updateTrigger,
  };
}
