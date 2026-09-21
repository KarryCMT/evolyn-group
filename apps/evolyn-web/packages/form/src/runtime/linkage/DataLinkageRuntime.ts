import type { DataLinkageDefinition, FormJsonValue } from '../../schema/types';
import type {
  FormRuntimeAdapter,
  LinkageExecuteRequest,
  LinkageExecuteResult,
} from '../adapters/types';

export interface DataLinkageRuntimeOptions {
  rules: readonly DataLinkageDefinition[];
  formId: string;
  schemaVersion: number;
  adapter?: FormRuntimeAdapter;
  valueOf(fieldId: string): FormJsonValue | undefined;
  applyValues(values: Record<string, FormJsonValue>): void;
  clearTargets(rule: DataLinkageDefinition): void;
}

/**
 * 每个表单会话独立持有的联动调度器：依赖字段索引、逐规则防抖、AbortController
 * 与 requestVersion 共同保证旧响应永远不能覆盖新输入。
 */
export class DataLinkageRuntime {
  private readonly dependencies = new Map<string, Set<DataLinkageDefinition>>();
  private readonly timers = new Map<string, ReturnType<typeof setTimeout>>();
  private readonly controllers = new Map<string, AbortController>();
  private readonly versions = new Map<string, number>();

  constructor(private readonly options: DataLinkageRuntimeOptions) {
    for (const rule of options.rules) {
      if (!rule.enabled) continue;
      for (const condition of rule.filter.conditions) {
        if (condition.value?.type !== 'field') continue;
        const rules = this.dependencies.get(condition.value.fieldId) ?? new Set();
        rules.add(rule);
        this.dependencies.set(condition.value.fieldId, rules);
      }
    }
  }

  initialize(): void {
    for (const rule of this.options.rules) {
      if (rule.enabled && rule.runtime.runOnInit) this.schedule(rule);
    }
  }

  notifyFieldChange(fieldId: string): void {
    for (const rule of this.dependencies.get(fieldId) ?? []) this.schedule(rule);
  }

  dispose(): void {
    for (const timer of this.timers.values()) clearTimeout(timer);
    for (const controller of this.controllers.values()) controller.abort();
    this.timers.clear();
    this.controllers.clear();
  }

  private schedule(rule: DataLinkageDefinition): void {
    const current = this.timers.get(rule.id);
    if (current) clearTimeout(current);
    // 输入一变化就立即让在途响应失效；不能等到下一次防抖真正发请求时才递增版本，
    // 否则旧响应可能在防抖窗口内短暂覆盖用户的新输入。
    this.controllers.get(rule.id)?.abort();
    this.versions.set(rule.id, (this.versions.get(rule.id) ?? 0) + 1);
    this.timers.set(
      rule.id,
      setTimeout(() => void this.execute(rule), Math.max(0, rule.runtime.debounceMs)),
    );
  }

  private async execute(rule: DataLinkageDefinition): Promise<void> {
    this.timers.delete(rule.id);
    const executeLinkage = this.options.adapter?.executeLinkage;
    if (!executeLinkage || !this.options.formId || this.options.schemaVersion < 1) return;

    // 任一当前字段依赖为空时跳过网络查询，直接执行无结果策略。
    const fieldDependencies = rule.filter.conditions
      .map((condition) => (condition.value?.type === 'field' ? condition.value.fieldId : ''))
      .filter(Boolean);
    if (fieldDependencies.some((field) => isEmpty(this.options.valueOf(field)))) {
      if (rule.runtime.emptyStrategy === 'clear') this.options.clearTargets(rule);
      return;
    }

    this.controllers.get(rule.id)?.abort();
    const controller = new AbortController();
    this.controllers.set(rule.id, controller);
    const requestVersion = this.versions.get(rule.id) ?? 1;
    const values: Record<string, FormJsonValue> = {};
    for (const field of fieldDependencies) {
      const value = this.options.valueOf(field);
      if (value !== undefined) values[field] = value;
    }
    const request: LinkageExecuteRequest = {
      formId: this.options.formId,
      schemaVersion: this.options.schemaVersion,
      ruleId: rule.id,
      requestVersion,
      values,
    };

    try {
      const result: LinkageExecuteResult = await executeLinkage(request, controller.signal);
      if (controller.signal.aborted || this.versions.get(rule.id) !== result.requestVersion) return;
      if (!result.matched) {
        if (rule.runtime.emptyStrategy === 'clear') this.options.clearTargets(rule);
        return;
      }
      this.options.applyValues(result.values);
    } catch (error) {
      if (controller.signal.aborted || isAbortError(error)) return;
      if (rule.runtime.errorStrategy === 'clear') this.options.clearTargets(rule);
    }
  }
}

function isEmpty(value: FormJsonValue | undefined): boolean {
  return (
    value === undefined ||
    value === null ||
    value === '' ||
    (Array.isArray(value) && value.length === 0)
  );
}

function isAbortError(error: unknown): boolean {
  return Boolean(
    error &&
    typeof error === 'object' &&
    ((error as { name?: string }).name === 'AbortError' ||
      (error as { code?: string }).code === 'ERR_CANCELED'),
  );
}
