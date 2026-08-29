import type { ShallowRef } from 'vue';
import type { FormDraftPayload, FormSubmitPayload } from '../types';
import type { FormDraftOutcome, FormRuntime, FormSubmitOutcome } from '../store/createFormRuntime';

/** Surface 等可信宿主可调用的最小命令接口；字段值仍只能通过运行时 action 修改。 */
export interface FormRendererExpose {
  readonly runtime: Readonly<ShallowRef<FormRuntime | null>>;
  getRuntime(): FormRuntime | null;
  submit(): Promise<FormSubmitOutcome | undefined>;
  saveDraft(): Promise<FormDraftOutcome | undefined>;
  buildSubmitPayload(): FormSubmitPayload | null;
  buildDraftPayload(): FormDraftPayload | null;
  focusFirstError(): boolean;
  reset(): void;
}
