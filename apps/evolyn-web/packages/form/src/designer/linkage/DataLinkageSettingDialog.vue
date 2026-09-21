<script setup lang="ts">
import type { DataLinkageDefinition, FormItem } from '../../schema/types';
import {
  type LinkageDesignerAdapter,
  type LinkageSourceField,
  cloneDataLinkageDefinition,
  createDataLinkageConditionId,
  createDataLinkageDefinition,
  linkageCurrentFields,
} from '../../schema/linkage';
import {
  ElButton,
  ElCheckbox,
  ElDialog,
  ElMessage,
  ElOption,
  ElRadio,
  ElRadioGroup,
  ElSelect,
} from 'element-plus';
import { computed, ref, shallowRef, watch } from 'vue';
import LinkageConditionEditor from './LinkageConditionEditor.vue';
import LinkageMappingEditor from './LinkageMappingEditor.vue';

const props = defineProps<{
  modelValue: boolean;
  rule?: DataLinkageDefinition;
  appId: number;
  targetFieldId: string;
  items: FormItem[];
  adapter?: LinkageDesignerAdapter;
}>();

const emit = defineEmits<{
  'update:modelValue': [visible: boolean];
  confirm: [rule: DataLinkageDefinition];
}>();

const draft = ref<DataLinkageDefinition>();
const sources = ref<Array<{ sourceId: string; appId: number; name: string }>>([]);
const sourceFields = ref<LinkageSourceField[]>([]);
const loadingSources = shallowRef(false);
const loadingFields = shallowRef(false);
const currentFields = computed(() => linkageCurrentFields(props.items));
let requestController: AbortController | undefined;

watch(
  () => props.modelValue,
  async (visible) => {
    if (!visible) {
      requestController?.abort();
      return;
    }
    draft.value = props.rule
      ? cloneDataLinkageDefinition(props.rule)
      : createDataLinkageDefinition(props.appId, props.targetFieldId);
    await loadSources();
    if (draft.value.source.sourceId) await loadSourceFields(draft.value.source.sourceId);
  },
);

async function loadSources(): Promise<void> {
  if (!props.adapter) return;
  requestController?.abort();
  requestController = new AbortController();
  loadingSources.value = true;
  try {
    sources.value = await props.adapter.listSources(requestController.signal);
  } catch {
    if (!requestController.signal.aborted) ElMessage.error('联动表单加载失败，请稍后重试');
  } finally {
    loadingSources.value = false;
  }
}

async function selectSource(sourceId: string): Promise<void> {
  if (!draft.value) return;
  draft.value.source.sourceId = sourceId;
  const source = sources.value.find((entry) => entry.sourceId === sourceId);
  if (source) draft.value.source.appId = source.appId;
  draft.value.filter.conditions.forEach((condition) => (condition.sourceFieldId = ''));
  draft.value.mappings.forEach((mapping) => (mapping.sourceFieldId = ''));
  await loadSourceFields(sourceId);
}

async function loadSourceFields(sourceId: string): Promise<void> {
  if (!props.adapter) return;
  requestController?.abort();
  requestController = new AbortController();
  loadingFields.value = true;
  try {
    sourceFields.value = await props.adapter.listSourceFields(sourceId, requestController.signal);
  } catch {
    if (!requestController.signal.aborted) ElMessage.error('联动字段加载失败，请稍后重试');
  } finally {
    loadingFields.value = false;
  }
}

function addCondition(): void {
  draft.value?.filter.conditions.push({
    id: createDataLinkageConditionId(),
    sourceFieldId: '',
    operator: 'eq',
    value: { type: 'field', fieldId: props.targetFieldId },
  });
}

function addMapping(): void {
  draft.value?.mappings.push({ sourceFieldId: '', targetFieldId: '' });
}

function confirm(): void {
  const rule = draft.value;
  if (!rule?.source.sourceId) return void ElMessage.warning('请选择联动表单');
  if (
    !rule.filter.conditions.length ||
    rule.filter.conditions.some(
      (condition) =>
        !condition.sourceFieldId ||
        (!['empty', 'not_empty'].includes(condition.operator) &&
          (!condition.value || (condition.value.type === 'field' && !condition.value.fieldId))),
    )
  ) {
    return void ElMessage.warning('请设置完整的数据联动条件');
  }
  if (
    !rule.mappings.length ||
    rule.mappings.some((mapping) => !mapping.sourceFieldId || !mapping.targetFieldId)
  ) {
    return void ElMessage.warning('请至少设置一条完整的联动映射');
  }
  if (
    new Set(rule.mappings.map((mapping) => mapping.targetFieldId)).size !== rule.mappings.length
  ) {
    return void ElMessage.warning('同一当前表单字段只能设置一次联动映射');
  }
  emit('confirm', cloneDataLinkageDefinition(rule));
  emit('update:modelValue', false);
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    class="data-linkage-dialog"
    title="数据联动设置"
    width="min(1080px, 92vw)"
    top="6vh"
    append-to-body
    destroy-on-close
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="draft" class="data-linkage-dialog__body">
      <section class="data-linkage-dialog__section">
        <h3>联动表单</h3>
        <el-select
          :model-value="draft.source.sourceId"
          filterable
          :loading="loadingSources"
          placeholder="请选择联动表单"
          aria-label="联动表单"
          @update:model-value="selectSource(String($event))"
        >
          <el-option
            v-for="source in sources"
            :key="source.sourceId"
            :label="source.name"
            :value="source.sourceId"
          />
        </el-select>
      </section>

      <section class="data-linkage-dialog__section">
        <div class="data-linkage-dialog__condition-heading">
          <span class="data-linkage-dialog__required">*</span>
          <span>联动表单字段满足以下</span>
          <el-select
            v-model="draft.filter.logic"
            class="data-linkage-dialog__logic"
            aria-label="条件关系"
          >
            <el-option label="所有" value="and" />
            <el-option label="任一" value="or" />
          </el-select>
          <span>条件时</span>
        </div>
        <el-button type="primary" link class="data-linkage-dialog__add" @click="addCondition">
          ＋ 添加过滤条件
        </el-button>
        <LinkageConditionEditor
          v-model="draft.filter.conditions"
          :source-fields="sourceFields"
          :current-fields="currentFields"
          :loading="loadingFields"
        />
      </section>

      <section class="data-linkage-dialog__section">
        <h3>触发以下联动</h3>
        <el-button type="primary" link class="data-linkage-dialog__add" @click="addMapping">
          ＋ 添加联动字段
        </el-button>
        <LinkageMappingEditor
          v-model="draft.mappings"
          :source-fields="sourceFields"
          :current-fields="currentFields"
          :loading="loadingFields"
        />
      </section>

      <section class="data-linkage-dialog__options">
        <el-checkbox v-model="draft.runtime.runOnInit">表单初始化时执行</el-checkbox>
        <span>无匹配数据时</span>
        <el-radio-group v-model="draft.runtime.emptyStrategy">
          <el-radio value="clear">清空联动字段</el-radio>
          <el-radio value="keep">保持原值</el-radio>
        </el-radio-group>
      </section>
    </div>

    <template #footer>
      <div class="data-linkage-dialog__footer">
        <a
          href="javascript:void(0)"
          @click.prevent="
            ElMessage.info('数据联动会按当前字段值查询已发布表单数据，并自动回填映射字段。')
          "
        >什么是数据联动？</a>
        <div>
          <el-button @click="emit('update:modelValue', false)">取消</el-button>
          <el-button type="primary" @click="confirm">确定</el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<style lang="scss">
.data-linkage-dialog {
  border-radius: var(--el-border-radius-large);

  .el-dialog__header {
    padding: var(--el-space-xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .el-dialog__body {
    min-height: 520px;
    padding: var(--el-space-xl);
  }

  &__body,
  &__section {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-md);
  }

  &__body {
    gap: var(--el-space-xl);
  }

  &__section h3 {
    margin: 0;
    font-size: var(--el-font-size-medium);
  }

  &__condition-heading {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
  }

  &__required {
    color: var(--el-color-danger);
  }

  &__logic {
    width: 92px;
  }

  &__add {
    align-self: flex-start;
    padding-left: 0;
  }

  &__options {
    display: flex;
    gap: var(--el-space-lg);
    align-items: center;
  }

  &__footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
  }

  &__footer a {
    color: var(--el-color-primary);
  }
}
</style>
