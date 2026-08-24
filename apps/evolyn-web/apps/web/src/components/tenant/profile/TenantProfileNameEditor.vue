<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue';

const props = defineProps<{
  modelValue: string;
  visible: boolean;
}>();

const emit = defineEmits<{
  cancel: [];
  save: [name: string];
}>();

const draftName = shallowRef(props.modelValue);
const errorMessage = computed(() => {
  const name = draftName.value.trim();
  if (!name) return '企业名称不能为空';
  if (name.length > 50) return '企业名称不能超过 50 个字符';
  return '';
});

watch(
  () => props.visible,
  (visible) => {
    if (visible) draftName.value = props.modelValue;
  },
);

watch(
  () => props.modelValue,
  (name) => {
    if (props.visible) draftName.value = name;
  },
);

function save() {
  if (errorMessage.value) return;
  emit('save', draftName.value.trim());
}
</script>

<template>
  <div v-if="props.visible" class="tenant-profile-name-editor">
    <div class="tenant-profile-name-editor__rules" role="status">
      <p><span>✓</span> 企业名称不能为空</p>
      <p><span>✓</span> 不能超过 50 个字符</p>
    </div>
    <div class="tenant-profile-name-editor__controls">
      <el-input
        v-model="draftName"
        class="tenant-profile-name-editor__input"
        :class="{ 'tenant-profile-name-editor__input--error': errorMessage }"
        maxlength="50"
        autofocus
        @keyup.enter="save"
      />
      <el-button type="primary" :disabled="Boolean(errorMessage)" @click="save">确定</el-button>
      <el-button @click="emit('cancel')">取消</el-button>
    </div>
    <p v-if="errorMessage" class="tenant-profile-name-editor__error">{{ errorMessage }}</p>
  </div>
</template>

<style scoped lang="scss">
.tenant-profile-name-editor {
  position: relative;
  width: min(100%, 940px);
  padding-top: 80px;

  &__rules {
    position: absolute;
    top: 0;
    left: 0;
    box-sizing: border-box;
    width: min(100%, 620px);
    padding: 16px 28px;
    border-radius: 10px;
    background: var(--el-bg-color-overlay);
    box-shadow: var(--el-box-shadow-light);

    &::after {
      position: absolute;
      bottom: -8px;
      left: 22px;
      width: 16px;
      height: 16px;
      background: var(--el-bg-color-overlay);
      content: '';
      transform: rotate(45deg);
    }

    p {
      position: relative;
      z-index: 1;
      display: flex;
      margin: 0;
      align-items: center;
      gap: 10px;
      color: var(--el-text-color-regular);
      font-size: 14px;
      line-height: 24px;
    }

    span {
      display: inline-flex;
      width: 18px;
      height: 18px;
      border-radius: 50%;
      align-items: center;
      justify-content: center;
      color: var(--el-color-white);
      background: var(--el-color-success);
      font-size: 12px;
      font-weight: 700;
    }
  }

  &__controls {
    display: flex;
    align-items: flex-start;
    gap: 12px;
  }

  &__input {
    flex: 1;

    :deep(.el-input__wrapper) {
      min-height: 40px;
    }

    &--error :deep(.el-input__wrapper) {
      box-shadow: 0 0 0 1px var(--el-color-danger) inset;
    }
  }

  &__error {
    margin: 6px 0 0;
    color: var(--el-color-danger);
    font-size: 13px;
    line-height: 20px;
  }
}

@media (max-width: 640px) {
  .tenant-profile-name-editor {
    padding-top: 106px;

    &__controls {
      flex-wrap: wrap;
    }

    &__input {
      width: 100%;
      flex-basis: 100%;
    }
  }
}
</style>
