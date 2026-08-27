<template>
  <!-- 插件代码与参数 JSON 视图，参数 JSON 校验成功后同步回表单配置。 -->
  <div class="plugin-design-code">
    <div class="plugin-design-code__editor">
      <FormCodeEditor
        ref="codeEditorRef"
        :model-value="editorContent"
        :language="editorLanguage"
        :hide-horizontal-scrollbar="showSchemaPanel && schemaCollapsed"
        :diagnostics="editorDiagnostics"
        :diagnostic-focus-key="diagnosticFocusKey"
        @update:model-value="updateEditorContent"
      />
    </div>
    <div
      v-if="showSchemaPanel"
      class="plugin-design-code__schema-panel"
      :class="{ 'is-collapsed': schemaCollapsed }"
    >
      <button
        class="plugin-design-code__schema-toggle"
        type="button"
        @click="schemaCollapsed = !schemaCollapsed"
      >
        <el-icon>
          <DArrowLeft v-if="schemaCollapsed" />
          <DArrowRight v-else />
        </el-icon>
      </button>
      <div class="plugin-design-code__schema-clip">
        <aside class="plugin-design-code__schema">
          <div class="plugin-design-code__schema-title">{{ t('通用参数') }}</div>
          <div
            v-for="field in globalFields"
            :key="field.fieldKey"
            class="plugin-design-code__schema-row"
          >
            <span>{{ field.fieldKey }}</span>
            <em>{{ field.widgetName }}</em>
          </div>
          <div class="plugin-design-code__schema-title">{{ t('请求参数') }}</div>
          <div
            v-for="field in functionData.fields"
            :key="field.fieldKey"
            class="plugin-design-code__schema-row"
          >
            <span>{{ field.fieldKey }}</span>
            <em>{{ field.widgetName }}</em>
          </div>
          <div class="plugin-design-code__schema-title">{{ t('返回参数') }}</div>
          <div
            v-for="item in responseSchemaFields"
            :key="item.field.id || `${item.level}_${item.field.fieldKey}`"
            class="plugin-design-code__schema-row"
            :class="{ 'is-child': item.level > 0 }"
            :style="{ '--plugin-design-response-level': item.level }"
          >
            <span>{{ item.field.fieldKey }}</span>
            <em>{{ item.field.dataType }}</em>
          </div>
        </aside>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElIcon } from 'element-plus';
import { computed, ref } from 'vue';
import { DArrowLeft, DArrowRight } from '@element-plus/icons-vue';
import type {
  PluginCodeDiagnostic,
  PluginDesignField,
  PluginDesignFunction,
  PluginDesignMenuKey,
  PluginDesignResponseField,
} from '../types';
import { validateFormCode } from '../hooks/useFormCodeValidation';
import { useFormDesignJsonEditor } from '../hooks/useFormDesignJsonEditor';
import { getFormRuntimeLanguage } from '../hooks/useFormRuntime';
import FormCodeEditor from './FormCodeEditor.vue';

const props = defineProps<{
  functionData: PluginDesignFunction;
  globalFields: PluginDesignField[];
  activeMenu: PluginDesignMenuKey;
  diagnostics?: PluginCodeDiagnostic[];
  diagnosticFocusKey?: number;
}>();

const t = (text: string) => text;
const schemaCollapsed = ref(false);
// 暴露编辑器语言服务的语法诊断，供主面板在保存前统一校验。
const codeEditorRef = ref<{
  getSyntaxDiagnostics: () => PluginCodeDiagnostic[];
}>();
const showSchemaPanel = computed(() => props.activeMenu === 'code');
const isConfigCodeView = computed(() =>
  ['common', 'request', 'response'].includes(props.activeMenu),
);

/**
 * 将返回参数树转换为带层级信息的列表，供代码页概览完整展示任意层级的 vector 子参数。
 */
const responseSchemaFields = computed(() => {
  const fields: Array<{ field: PluginDesignResponseField; level: number }> = [];
  const appendFields = (responseFields: PluginDesignResponseField[], level: number) => {
    responseFields.forEach((field) => {
      fields.push({ field, level });
      if (Array.isArray(field.fieldConf.fields)) appendFields(field.fieldConf.fields, level + 1);
    });
  };
  appendFields(props.functionData.responseParams, 0);
  return fields;
});

const editorDiagnostics = computed(() => {
  if (props.activeMenu !== 'code') return [];
  return props.diagnostics?.length ? props.diagnostics : validateFormCode(props.functionData);
});
const editorLanguage = computed<'javascript' | 'python' | 'java' | 'json'>(() => {
  if (isConfigCodeView.value) return 'json';
  // 按运行时枚举匹配 Monaco 语言，旧数据由工具函数兼容归一。
  return getFormRuntimeLanguage(props.functionData.runtime);
});
const { editorContent, refreshEditorContent, updateEditorContent, validateConfigDraft } =
  useFormDesignJsonEditor({
    getActiveMenu: () => props.activeMenu,
    getFunctionData: () => props.functionData,
    getGlobalFields: () => props.globalFields,
    t,
  });

/** 仅在函数源码页读取 Monaco 语法错误，参数 JSON 继续使用结构化草稿校验。 */
const getSyntaxDiagnostics = () => {
  if (props.activeMenu !== 'code') return [];
  return codeEditorRef.value?.getSyntaxDiagnostics() || [];
};

// 主面板在保存、切换菜单和切换视图前调用校验，避免非法 JSON 草稿被静默丢弃。
defineExpose({
  getSyntaxDiagnostics,
  refreshEditorContent,
  validateConfigDraft,
});
</script>

<style lang="scss" scoped>
.plugin-design-code {
  display: flex;
  height: 100%;
  min-height: 0;
  overflow: hidden;

  &__editor {
    flex: 1;
    min-width: 0;
    background-color: #1e1e1e;
    border-radius: var(--gp-radius-xl);
  }

  &__schema-panel {
    position: relative;
    flex: 0 0 auto;
    width: 280px;
    transition: width 0.22s ease;

    &.is-collapsed {
      width: 0;

      .plugin-design-code__schema {
        opacity: 0;
        transform: translateX(var(--el-space-md));
      }
    }
  }

  &__schema-toggle {
    position: absolute;
    top: var(--gp-space-xl);
    left: -24px;
    z-index: 2;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 32px;
    padding: 0;
    color: var(--el-text-color-primary);
    cursor: pointer;
    background-color: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-right: 0;
    border-radius: var(--el-border-radius-base) 0 0 var(--el-border-radius-base);
    box-shadow: var(--el-box-shadow);

    .el-icon {
      font-size: var(--el-font-size-medium);
    }
  }

  &__schema-clip {
    width: 100%;
    height: 100%;
    overflow: hidden;
    background-color: var(--el-bg-color);
    border-left: 1px solid var(--el-border-color-lighter);
  }

  &__schema {
    box-sizing: border-box;
    width: 280px;
    height: 100%;
    padding: 0 var(--gp-space-xl) 0 var(--gp-space-xl);
    overflow: auto;
    background-color: var(--el-fill-color-light);
    opacity: 1;
    transition:
      opacity 0.16s ease,
      transform 0.22s ease;
  }

  &__schema-title {
    margin-bottom: var(--gp-space-lg);
    font-size: var(--el-font-size-extra-small);
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__schema-row {
    display: flex;
    justify-content: space-between;
    margin-bottom: var(--gp-space-sm);
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-primary);

    &.is-child {
      padding-left: calc(var(--plugin-design-response-level) * var(--gp-space-lg));
    }

    em {
      font-style: normal;
      color: var(--el-text-color-secondary);
    }
  }
}
</style>
