import { Numeric } from '@evolyn.do/numeric';
import { evaluateFormula } from '@evolyn.do/formula-runtime';
import { type FormulaNode, parseFormula } from '@evolyn.do/formula';
import {
  type FieldFormulaDefinition,
  type FormJsonValue,
  formulaDependencies,
  sortFieldFormulas,
} from '../../schema';

interface CompiledFieldFormula {
  definition: FieldFormulaDefinition;
  ast: FormulaNode;
  dependencies: readonly string[];
}

export interface FieldFormulaRuntimeOptions {
  formulas: readonly FieldFormulaDefinition[];
  valueOf: (fieldId: string) => FormJsonValue | undefined;
  applyValue: (fieldId: string, value: FormJsonValue) => void;
  onSuccess?: (fieldId: string) => void;
  onError?: (fieldId: string, message: string) => void;
}

/**
 * 字段公式会话执行器：初始化按拓扑全量计算，字段变化只执行受影响的下游公式。
 * 它不持有 Vue 状态，所有值读写均经 Runtime Store 注入，便于 Web/移动端共用。
 */
export class FieldFormulaRuntime {
  readonly #compiled: readonly CompiledFieldFormula[];
  readonly #downstream = new Map<string, Set<string>>();
  readonly #byTarget = new Map<string, CompiledFieldFormula>();

  constructor(private readonly options: FieldFormulaRuntimeOptions) {
    const { ordered } = sortFieldFormulas(options.formulas);
    const compiled: CompiledFieldFormula[] = [];
    for (const definition of ordered) {
      const parsed = parseFormula(definition.formula);
      if (!parsed.ast || parsed.diagnostics.length > 0) continue;
      const entry = {
        definition,
        ast: parsed.ast,
        dependencies: formulaDependencies(definition.formula),
      };
      compiled.push(entry);
      this.#byTarget.set(definition.targetFieldId, entry);
      for (const dependency of entry.dependencies) {
        const targets = this.#downstream.get(dependency) ?? new Set<string>();
        targets.add(definition.targetFieldId);
        this.#downstream.set(dependency, targets);
      }
    }
    this.#compiled = compiled;
  }

  initialize(): void {
    for (const formula of this.#compiled) this.#evaluate(formula);
  }

  notifyFieldChange(fieldId: string): void {
    const affected = this.#collectAffected(fieldId);
    if (affected.size === 0) return;
    for (const formula of this.#compiled) {
      if (affected.has(formula.definition.targetFieldId)) this.#evaluate(formula);
    }
  }

  #collectAffected(fieldId: string): Set<string> {
    const affected = new Set<string>();
    const queue = [...(this.#downstream.get(fieldId) ?? [])];
    while (queue.length > 0) {
      const target = queue.shift()!;
      if (affected.has(target)) continue;
      affected.add(target);
      queue.push(...(this.#downstream.get(target) ?? []));
    }
    return affected;
  }

  #evaluate(compiled: CompiledFieldFormula): void {
    try {
      const value = evaluateFormula(compiled.ast, {
        source: compiled.definition.formula,
        resolveField: (fieldId) => this.options.valueOf(fieldId) as never,
      });
      this.options.applyValue(compiled.definition.targetFieldId, serializeFormulaValue(value));
      this.options.onSuccess?.(compiled.definition.targetFieldId);
    } catch (error) {
      const message = error instanceof Error ? error.message : '公式计算失败';
      this.options.onError?.(compiled.definition.targetFieldId, message);
      this.options.applyValue(compiled.definition.targetFieldId, null);
    }
  }
}

function serializeFormulaValue(value: ReturnType<typeof evaluateFormula>): FormJsonValue {
  if (value instanceof Numeric) return value.serialize();
  if (Array.isArray(value)) {
    return value.map((entry) =>
      entry instanceof Numeric ? entry.serialize() : (entry as FormJsonValue),
    );
  }
  return value;
}
