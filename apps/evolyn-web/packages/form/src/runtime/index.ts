// 运行时入口：只允许静态依赖 Vue、Schema（@evolyn.do/form/schema）、运行时 Store、
// 校验与基础字段。富文本、签名、附件、成员/部门选择器等重型字段必须以 loader
// 动态注册（registry.register('user', () => import('./widgets/heavy/UserField.vue'))），
// 不得在此入口静态 import，否则会被合并进首屏产物。
// 样式经独立入口 @evolyn.do/form/runtime/style.css 引入，保持首屏 CSS 最小化。

export type {
  FieldRuntimeState,
  FormDraftPayload,
  FormIssue,
  FormRuntimeLifecycle,
  FormRuntimeOperation,
  FormRuntimeState,
  FormSubmitPayload,
  FormSubmitResult,
  FormValue,
  FormValueSource,
  RuntimeFieldEmits,
  RuntimeFieldProps,
} from './types';
export type {
  DepartmentQuery,
  DepartmentValue,
  FileValue,
  FormRuntimeAdapter,
  MemberQuery,
  MemberValue,
  Page,
  RelatedDataQuery,
  RelatedValue,
  UploadInput,
} from './adapters/types';
export { createFormRuntime } from './store/createFormRuntime';
export type {
  FormDraftOutcome,
  FormRuntime,
  FormRuntimeOptions,
  FormSubmitOutcome,
} from './store/createFormRuntime';
export type {
  FormRuntimeActionBehavior,
  FormRuntimeActionDefinition,
  FormRuntimeActionIntent,
  FormRuntimeLayout,
  FormRuntimeMobilePresentation,
} from './actions/types';
export {
  FormRendererContextKey,
  useFormRendererContext,
  type FormRendererContext,
} from './store/injection';
export { buildRenderPlan } from './renderer/plan';
export type {
  FormRenderFieldNode,
  FormRenderMultitabNode,
  FormRenderNode,
  FormRenderPlan,
  FormRenderSection,
  FormRenderTab,
} from './renderer/plan';
export { default as FormRenderer } from './renderer/FormRenderer.vue';
export type { FormRendererExpose } from './renderer/types';
export { default as FormSectionRenderer } from './renderer/FormSectionRenderer.vue';
export { default as FormFieldHost } from './renderer/FormFieldHost.vue';
export { default as FormFieldError } from './renderer/FormFieldError.vue';
export { default as FormRuntimeActionBar } from './surface/FormRuntimeActionBar.vue';
export { default as FormRuntimeSurface } from './surface/FormRuntimeSurface.vue';
export {
  createDefaultFieldRegistry,
  FormFieldRegistry,
  type FieldWidgetDefinition,
  type FieldWidgetLoader,
} from './widgets/registry';
// 以下助手面向自定义字段开发者导出，保证扩展字段与内置字段遵循同一值协议
//（语义定义在 schema 层，前后端一致）。
export {
  emptyWidgetValue,
  isEmptyWidgetValue,
  isLayoutWidgetType,
  isMultiValueWidgetType,
  normalizeWidgetValue,
  readWidgetBooleanConfig,
  readWidgetNumberConfig,
  readWidgetOptions,
  readWidgetStringConfig,
  validateWidgetValue,
} from '../schema/codec';
export {
  fieldAriaDescribedBy,
  fieldDescriptionId,
  fieldErrorId,
  fieldInputId,
  fieldLabelId,
} from './field-dom';
