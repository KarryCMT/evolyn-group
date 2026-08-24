<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue';
import {
  cloneWorkflowDocument,
  type WorkflowDocument,
  type WorkflowField,
  type WorkflowNode,
} from '../schema';
import WorkflowCanvas from './WorkflowCanvas.vue';
import WorkflowInspector from './WorkflowInspector.vue';

defineOptions({ name: 'WorkflowDesigner' });

const props = defineProps<{
  document: WorkflowDocument;
  fields: readonly WorkflowField[];
}>();

const emit = defineEmits<{
  updateDocument: [document: WorkflowDocument];
}>();

const selectedNodeId = shallowRef<string | null>(props.document.nodes[0]?.id ?? null);
const selectedNode = computed(
  () => props.document.nodes.find((node) => node.id === selectedNodeId.value) ?? null,
);

watch(
  () => props.document.nodes,
  (nodes) => {
    if (!nodes.some((node) => node.id === selectedNodeId.value)) {
      selectedNodeId.value = nodes[0]?.id ?? null;
    }
  },
);

function selectNode(nodeId: string) {
  selectedNodeId.value = nodeId;
}

function updateNode(nodeId: string, patch: Partial<WorkflowNode>) {
  const nextDocument = cloneWorkflowDocument(props.document);
  const target = nextDocument.nodes.find((node) => node.id === nodeId);
  if (!target) return;

  Object.assign(target, patch);
  if (patch.position) target.position = { ...patch.position };
  if (patch.fieldPermissions) {
    target.fieldPermissions = Object.fromEntries(
      Object.entries(patch.fieldPermissions).map(([fieldId, permission]) => [
        fieldId,
        { ...permission },
      ]),
    );
  }
  if (patch.config) target.config = { ...patch.config };
  emit('updateDocument', nextDocument);
}

function updateNodePosition(nodeId: string, position: WorkflowNode['position']) {
  updateNode(nodeId, { position });
}
</script>

<template>
  <section class="workflow-designer" aria-label="流程设计器">
    <WorkflowCanvas
      class="workflow-designer__canvas"
      :document="document"
      :selected-node-id="selectedNodeId"
      @select-node="selectNode"
      @update-node-position="updateNodePosition"
    />
    <WorkflowInspector
      class="workflow-designer__inspector"
      :fields="fields"
      :node="selectedNode"
      @update-node="updateNode"
    />
  </section>
</template>

<style scoped lang="scss">
.workflow-designer {
  display: grid;
  min-height: 0;
  flex: 1;
  grid-template-columns: minmax(0, 1fr) 360px;

  &__canvas,
  &__inspector {
    min-height: 0;
  }
}

@media (max-width: 900px) {
  .workflow-designer {
    grid-template-columns: minmax(0, 1fr) 300px;
  }
}

@media (max-width: 700px) {
  .workflow-designer {
    grid-template-columns: minmax(0, 1fr);

    &__inspector {
      display: none;
    }
  }
}
</style>
