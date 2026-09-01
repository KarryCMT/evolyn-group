<script setup lang="ts">
import type { ApplicationWorkspaceAsset } from './applicationWorkspace.types';
import { RiApps2Fill, RiArrowDownSFill, RiCloseFill, RiFolder3Fill } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';

defineOptions({ name: 'MoveMenuEntryDialog' });

const props = defineProps<{
  applicationName: string;
  assets: ApplicationWorkspaceAsset[];
  sourceAsset: ApplicationWorkspaceAsset;
  /** 返回 true 时弹窗向上通知保存成功，由页面层清理当前移动目标。 */
  submit: (parentEntryCode: string) => Promise<boolean>;
}>();

const emit = defineEmits<{
  success: [];
}>();

const ROOT_ENTRY_CODE = '';

interface TargetGroup {
  code: string;
  label: string;
  depth: number;
}

const visible = defineModel<boolean>({ default: false });
/** null 表示尚未选择；空字符串是服务端约定的应用根级。 */
const selectedGroupCode = shallowRef<string | null>(null);
const submitting = shallowRef(false);

/** 收集当前节点及其后代，避免将分组移动到自身或子分组下形成循环。 */
function collectSubtreeCodes(asset: ApplicationWorkspaceAsset, codes: Set<string>) {
  codes.add(asset.code);
  for (const child of asset.children ?? []) {
    collectSubtreeCodes(child, codes);
  }
}

const targetGroups = computed<TargetGroup[]>(() => {
  const excludedCodes = new Set<string>();
  if (props.sourceAsset.type === 'folder') {
    collectSubtreeCodes(props.sourceAsset, excludedCodes);
  }

  const groups: TargetGroup[] = [];
  const visit = (assets: ApplicationWorkspaceAsset[], depth: number) => {
    for (const asset of assets) {
      if (asset.type !== 'folder') continue;
      // 分组最多两级，因此分组只能移动到根级分组下；资产可以移动到任意分组。
      const isAllowedDepth = props.sourceAsset.type !== 'folder' || depth === 0;
      if (isAllowedDepth && !excludedCodes.has(asset.code)) {
        groups.push({ code: asset.code, label: asset.label, depth });
      }
      visit(asset.children ?? [], depth + 1);
    }
  };

  visit(props.assets, 0);
  return groups;
});

// 每次打开从未选中状态开始；取消不会改变应用菜单。
watch(visible, (isVisible) => {
  if (isVisible) selectedGroupCode.value = null;
});

function selectTarget(code: string) {
  if (!submitting.value) selectedGroupCode.value = code;
}

async function confirm() {
  if (selectedGroupCode.value === null || submitting.value) return;

  submitting.value = true;
  try {
    if (await props.submit(selectedGroupCode.value)) emit('success');
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="move-menu-entry-dialog"
    width="560px"
    top="20vh"
    :show-close="false"
    :close-on-click-modal="false"
    :close-on-press-escape="!submitting"
    append-to-body
  >
    <template #header>
      <header class="move-menu-entry-dialog__header">
        <h2 class="move-menu-entry-dialog__heading">移动</h2>
        <button
          class="move-menu-entry-dialog__close"
          type="button"
          aria-label="关闭移动弹窗"
          :disabled="submitting"
          @click="visible = false"
        >
          <RiCloseFill aria-hidden="true" />
        </button>
      </header>
    </template>

    <section class="move-menu-entry-dialog__body" aria-labelledby="move-menu-entry-prompt">
      <p id="move-menu-entry-prompt" class="move-menu-entry-dialog__prompt">请选择目标位置</p>

      <div class="move-menu-entry-dialog__tree" role="tree" aria-label="目标分组">
        <button
          class="move-menu-entry-dialog__application"
          :class="{
            'move-menu-entry-dialog__application--selected': selectedGroupCode === ROOT_ENTRY_CODE,
          }"
          type="button"
          role="treeitem"
          :aria-selected="selectedGroupCode === ROOT_ENTRY_CODE"
          :disabled="submitting"
          @click="selectTarget(ROOT_ENTRY_CODE)"
        >
          <RiArrowDownSFill class="move-menu-entry-dialog__expand-icon" aria-hidden="true" />
          <span class="move-menu-entry-dialog__application-icon" aria-hidden="true">
            <RiApps2Fill />
          </span>
          <span class="move-menu-entry-dialog__group-name"
            >{{ props.applicationName }}（根目录）</span
          >
          <span
            v-if="selectedGroupCode === ROOT_ENTRY_CODE"
            class="move-menu-entry-dialog__selected-mark"
            aria-label="已选中"
          >
            ✓
          </span>
        </button>

        <div class="move-menu-entry-dialog__groups" role="group">
          <button
            v-for="group in targetGroups"
            :key="group.code"
            class="move-menu-entry-dialog__group"
            :class="{ 'move-menu-entry-dialog__group--selected': selectedGroupCode === group.code }"
            :style="{ paddingLeft: `${44 + group.depth * 28}px` }"
            type="button"
            role="treeitem"
            :aria-selected="selectedGroupCode === group.code"
            :disabled="submitting"
            @click="selectTarget(group.code)"
          >
            <RiFolder3Fill class="move-menu-entry-dialog__folder-icon" aria-hidden="true" />
            <span class="move-menu-entry-dialog__group-name">{{ group.label }}</span>
            <span
              v-if="selectedGroupCode === group.code"
              class="move-menu-entry-dialog__selected-mark"
              aria-label="已选中"
            >
              ✓
            </span>
          </button>
        </div>
      </div>
    </section>

    <template #footer>
      <footer class="move-menu-entry-dialog__footer">
        <el-button :disabled="submitting" @click="visible = false"> 取消 </el-button>
        <el-button
          type="primary"
          :disabled="selectedGroupCode === null"
          :loading="submitting"
          @click="confirm"
        >
          确定
        </el-button>
      </footer>
    </template>
  </el-dialog>
</template>

<!-- 弹窗会传送至 body，故用唯一块类限制样式作用域。 -->
<style lang="scss">
.move-menu-entry-dialog.el-dialog {
  max-width: calc(100vw - 32px);
  margin-bottom: 0;
  overflow: hidden;
  border-radius: var(--el-border-radius-large);
}

.move-menu-entry-dialog .el-dialog__header,
.move-menu-entry-dialog .el-dialog__footer {
  padding: 0;
  margin: 0;
}

.move-menu-entry-dialog .el-dialog__header {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.move-menu-entry-dialog .el-dialog__body {
  padding: var(--el-space-2xl) var(--el-space-3xl);
}

.move-menu-entry-dialog .el-dialog__footer {
  border-top: 1px solid var(--el-border-color-lighter);
}

.move-menu-entry-dialog__header {
  display: flex;
  height: 56px;
  padding: 0 var(--el-space-lg) 0 var(--el-space-3xl);
  align-items: center;
  justify-content: space-between;
}

.move-menu-entry-dialog__heading,
.move-menu-entry-dialog__prompt {
  margin: 0;
  color: var(--el-text-color-primary);
}

.move-menu-entry-dialog__heading {
  font-size: var(--el-font-size-large);
  font-weight: 650;
}

.move-menu-entry-dialog__close {
  display: inline-flex;
  width: 32px;
  height: 32px;
  padding: 0;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--el-border-radius-base);
  font-size: 22px;

  &:hover:not(:disabled) {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }
}

.move-menu-entry-dialog__body {
  display: flex;
  min-height: 310px;
  flex-direction: column;
  gap: var(--el-space-xl);
}

.move-menu-entry-dialog__prompt {
  font-size: var(--el-font-size-medium);
  line-height: 1.5;
}

.move-menu-entry-dialog__tree {
  min-height: 250px;
  padding: var(--el-space-md);
  overflow: auto;
  border: 1px solid var(--el-border-color);
}

.move-menu-entry-dialog__application,
.move-menu-entry-dialog__group {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  min-height: 42px;
  align-items: center;
  gap: var(--el-space-md);
}

.move-menu-entry-dialog__application {
  padding: 0 var(--el-space-md);
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--el-border-radius-base);
  font-weight: 600;

  &:hover:not(:disabled),
  &--selected {
    background: var(--el-color-primary-light-9);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: -2px;
  }

  &:disabled {
    cursor: not-allowed;
  }
}

.move-menu-entry-dialog__expand-icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}

.move-menu-entry-dialog__application-icon {
  display: inline-flex;
  width: 24px;
  height: 24px;
  align-items: center;
  justify-content: center;
  color: var(--el-color-white);
  background: var(--el-color-primary);
  border-radius: var(--el-border-radius-small);
  font-size: 16px;
}

.move-menu-entry-dialog__group {
  padding-right: var(--el-space-md);
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--el-border-radius-base);

  &:hover:not(:disabled),
  &--selected {
    background: var(--el-color-primary-light-9);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: -2px;
  }

  &:disabled {
    cursor: not-allowed;
  }
}

.move-menu-entry-dialog__folder-icon {
  width: 24px;
  height: 24px;
  flex: 0 0 auto;
  color: #f5a623;
}

.move-menu-entry-dialog__group-name {
  min-width: 0;
  overflow: hidden;
  flex: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.move-menu-entry-dialog__selected-mark {
  display: inline-flex;
  width: 24px;
  height: 24px;
  align-items: center;
  justify-content: center;
  color: var(--el-color-white);
  background: var(--el-color-primary);
  border-radius: 50%;
  font-size: 16px;
  font-weight: 650;
}

.move-menu-entry-dialog__empty {
  padding: var(--el-space-xl) var(--el-space-md);
  margin: 0;
  color: var(--el-text-color-secondary);
  text-align: center;
}

.move-menu-entry-dialog__footer {
  display: flex;
  height: 64px;
  padding: 0 var(--el-space-3xl);
  align-items: center;
  justify-content: flex-end;
  gap: var(--el-space-md);
}

.move-menu-entry-dialog__footer .el-button {
  min-width: 72px;
  height: 34px;
  margin: 0;
}

@media (max-width: 600px) {
  .move-menu-entry-dialog.el-dialog {
    width: calc(100vw - 32px) !important;
  }

  .move-menu-entry-dialog .el-dialog__body {
    padding: var(--el-space-xl) var(--el-space-2xl);
  }

  .move-menu-entry-dialog__header,
  .move-menu-entry-dialog__footer {
    padding-right: var(--el-space-2xl);
    padding-left: var(--el-space-2xl);
  }
}
</style>
