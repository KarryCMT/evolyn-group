<script setup lang="ts">
import WorkspaceTitleEditor from '~/components/WorkspaceTitleEditor.vue';

defineOptions({ name: 'FormWorkspaceTitleEditor' });

defineProps<{
  name: string;
  disabled?: boolean;
  saving?: boolean;
}>();

const emit = defineEmits<{
  submit: [name: string, onSuccess: () => void];
}>();

function submit(name: string, onSuccess: () => void): void {
  emit('submit', name, onSuccess);
}
</script>

<template>
  <WorkspaceTitleEditor
    :name="name"
    resource-label="表单"
    :disabled="disabled"
    :saving="saving"
    @submit="submit"
  />
</template>
