import type {
  FormRuntimeAdapter,
  FormSubmitPayload,
  FormSubmitResult,
} from '@evolyn.do/form/runtime-mobile';
import type { MobileFormRuntimeBootstrap } from '~/types/app';
import { ApiError, http } from '@evolyn.do/utils';

export function getFormRuntime(
  appCode: string,
  formCode: string,
  signal?: AbortSignal,
): Promise<MobileFormRuntimeBootstrap> {
  return http.get(
    `/apps/code/${encodeURIComponent(appCode)}/forms/${encodeURIComponent(formCode)}/runtime`,
    undefined,
    signal,
  );
}

export function queryRelatedOptions(
  formCode: string,
  fieldId: string,
  payload: {
    schemaVersion: number;
    values: Record<string, unknown>;
    keyword?: string;
    pageSize?: number;
  },
  signal?: AbortSignal,
): Promise<{ items: Array<{ label: string; value: string }>; total: number }> {
  return http.post(
    `/forms/${encodeURIComponent(formCode)}/option-fields/${encodeURIComponent(fieldId)}/query`,
    payload,
    { signal },
  );
}

export function executeFormLinkage(
  formCode: string,
  ruleId: string,
  payload: {
    schemaVersion: number;
    values: Record<string, unknown>;
    requestVersion: number;
  },
  signal?: AbortSignal,
) {
  return http.post<Awaited<ReturnType<NonNullable<FormRuntimeAdapter['executeLinkage']>>>>(
    `/forms/${encodeURIComponent(formCode)}/linkages/${encodeURIComponent(ruleId)}/execute`,
    payload,
    { signal },
  );
}

/** 表单提交错误转换为运行时可定位的字段错误，不让页面层解析服务端结构。 */
export async function submitFormRecord(
  appCode: string,
  menuCode: string,
  payload: FormSubmitPayload,
  signal: AbortSignal,
): Promise<FormSubmitResult> {
  try {
    await http.post(
      '/form-records',
      {
        appCode,
        menuCode,
        formCode: payload.formId,
        publishedVersion: payload.publishedVersion,
        schemaRevision: payload.schemaRevision,
        values: payload.values,
        hasResult: true,
        dataOpId: payload.dataOpId ?? globalThis.crypto.randomUUID(),
      },
      { signal },
    );
    return { accepted: true };
  } catch (error) {
    if (
      error instanceof ApiError &&
      (error.errCode === 'FORM_RECORD_INVALID' ||
        error.errCode === 'FORM_RECORD_VALIDATION_FAILED')
    ) {
      const data = error.data as
        | {
            fieldErrors?: Record<string, string[]>;
            validatorErrors?: Array<{ index?: number; remind?: string; fields?: string[] }>;
          }
        | undefined;
      return {
        accepted: false,
        fieldErrors: data?.fieldErrors,
        validatorErrors: (data?.validatorErrors ?? []).flatMap((item, index) =>
          item.remind
            ? [{ index: item.index ?? index, remind: item.remind, fields: item.fields ?? [] }]
            : [],
        ),
        message: error.message,
      };
    }
    return {
      accepted: false,
      message:
        error instanceof ApiError && error.errCode === 'FORM_VERSION_CONFLICT'
          ? '表单已发布新版本，请返回后重新打开'
          : error instanceof Error
            ? error.message
            : '提交失败，请稍后重试',
    };
  }
}
