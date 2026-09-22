<script setup lang="ts">
import { ElButton, ElInput, ElMessage, ElOption, ElSelect } from 'element-plus';
import { computed, reactive, shallowRef, watch } from 'vue';
import {
  FIELD_FORMULA_FUNCTIONS,
  type FieldFormulaDefinition,
  type FormItem,
} from '../../schema';
import {
  FORMULA_FUNCTIONS,
  type FormulaEditorField,
  collectFormulaDiagnostics,
  projectFormulaContext,
} from '../../formula';
import FormFormulaEditorDialog from './FormFormulaEditorDialog.vue';

const open = defineModel<boolean>({ required: true });
const props = defineProps<{
  target: FormItem;
  items: readonly FormItem[];
  formula?: FieldFormulaDefinition;
}>();
const emit = defineEmits<{
  confirm: [formula: FieldFormulaDefinition];
}>();

const draft = reactive<FieldFormulaDefinition>(createDraft(props.target.widget.widgetName));
const replaceVisible = shallowRef(false);
const remarkVisible = shallowRef(false);
const replaceSource = shallowRef('');
const replaceTarget = shallowRef('');

const fields = computed<FormulaEditorField[]>(() =>
  projectFormulaContext(props.items).filter(
    (field) => field.formulaAllowed && field.widgetName !== props.target.widget.widgetName,
  ),
);
const functions = FORMULA_FUNCTIONS.filter((entry) => FIELD_FORMULA_FUNCTIONS.has(entry.name));
const diagnostics = computed(() =>
  draft.formula.trim()
    ? collectFormulaDiagnostics(draft.formula, fields.value, functions)
    : [],
);
const firstError = computed(() =>
  diagnostics.value.find((diagnostic) => diagnostic.severity === 'error'),
);

watch(open, (visible) => {
  if (!visible) return;
  Object.assign(
    draft,
    props.formula ? { ...props.formula } : createDraft(props.target.widget.widgetName),
  );
  replaceVisible.value = false;
  remarkVisible.value = false;
});

function confirm(): void {
  if (!draft.formula.trim()) {
    ElMessage.warning('请先编辑公式');
    return;
  }
  if (firstError.value) {
    ElMessage.warning(firstError.value.message);
    return;
  }
  emit('confirm', { ...draft });
  open.value = false;
  ElMessage.success('公式设置成功');
}

function replaceVariable(): void {
  if (!replaceSource.value || !replaceTarget.value || replaceSource.value === replaceTarget.value) {
    ElMessage.warning('请选择不同的原变量和目标变量');
    return;
  }
  draft.formula = draft.formula.split(`$${replaceSource.value}#`).join(`$${replaceTarget.value}#`);
  replaceVisible.value = false;
  replaceSource.value = '';
  replaceTarget.value = '';
}

function debugFormula(): void {
  if (!draft.formula.trim()) {
    ElMessage.warning('请先编辑公式');
    return;
  }
  if (firstError.value) {
    ElMessage.error(firstError.value.message);
    return;
  }
  ElMessage.success('公式结构检查通过');
}

function createDraft(targetFieldId: string): FieldFormulaDefinition {
  const suffix =
    globalThis.crypto?.randomUUID?.().replace(/-/g, '').slice(0, 12) ?? Date.now().toString(36);
  return {
    id: `formula_${suffix}`,
    version: 1,
    enabled: true,
    targetFieldId,
    formula: '',
    remark: '',
  };
}
</script>

<template>
  <FormFormulaEditorDialog
    v-model="open"
    v-model:formula="draft.formula"
    title="公式编辑"
    subtitle="使用字段和函数组合派生结果"
    :editor-label="target.label"
    :fields="fields"
    :functions="functions"
    :error="firstError?.message"
    footer-hint="字段值变化后将自动重新计算"
    aria-label="字段公式编辑器"
    @confirm="confirm"
  >
    <template #toolbar>
      <button type="button" @click="replaceVisible = !replaceVisible">替换变量</button>
      <button type="button" @click="remarkVisible = !remarkVisible">备注</button>
      <button type="button" @click="debugFormula">调试</button>
    </template>

    <template #editor-extra>
      <div v-if="replaceVisible" class="field-formula-setting__replace">
        <el-select v-model="replaceSource" placeholder="原变量" aria-label="待替换变量">
          <el-option
            v-for="field in fields"
            :key="field.widgetName"
            :label="field.label"
            :value="field.widgetName"
          />
        </el-select>
        <span>替换为</span>
        <el-select v-model="replaceTarget" placeholder="目标变量" aria-label="替换目标变量">
          <el-option
            v-for="field in fields"
            :key="field.widgetName"
            :label="field.label"
            :value="field.widgetName"
          />
        </el-select>
        <el-button type="primary" @click="replaceVariable">替换</el-button>
      </div>
      <div v-if="remarkVisible" class="field-formula-setting__remark">
        <el-input
          v-model="draft.remark"
          :maxlength="500"
          show-word-limit
          placeholder="填写公式用途或维护说明"
        />
      </div>
    </template>
  </FormFormulaEditorDialog>
</template>

<style scoped lang="scss">
.field-formula-setting__replace,
.field-formula-setting__remark {
  display: flex;
  gap: 10px;
  min-height: 52px;
  padding: 8px 14px;
  align-items: center;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.field-formula-setting__replace .el-select {
  width: 190px;
}

.field-formula-setting__remark .el-input {
  width: 100%;
}
</style>
