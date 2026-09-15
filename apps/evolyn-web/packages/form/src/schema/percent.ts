/**
 * 百分比字段的显示值与存储值转换。
 *
 * 协议和计算链统一把比例保存为 canonical decimal string：界面上的 `15%`
 * 在记录、公式与查询中均为 `0.15`。转换全程走 NumericRuntime，禁止经
 * Number/parseFloat 中转，以免在高精度比例上丢失精度。
 */
import { numeric } from '@evolyn.do/numeric';
import { DECIMAL_TEXT_PATTERN } from './numeric';

/**
 * v8 及更早发布快照没有该标记，历史百分比值按“填写值即存储值”读取，避免
 * `15` 被错误展示为 `1500%`。只有显式声明 ratio 的新字段才启用比例协议。
 */
export function usesPercentRatio(widget: { type?: unknown; percentValueMode?: unknown }): boolean {
  return widget.type === 'percent' && widget.percentValueMode === 'ratio';
}

/** 将持久化比例转为不带百分号的界面输入值，例如 `0.15` → `15`。 */
export function formatPercentRatio(value: string | null | undefined): string {
  if (!value || !DECIMAL_TEXT_PATTERN.test(value)) return value ?? '';
  try {
    return numeric.multiply(value, '100').serialize() ?? value;
  } catch {
    // 草稿里可能暂存尚未通过 schema 校验的旧值；保留原文方便设计者修正。
    return value;
  }
}

/** 将不带百分号的界面输入值转为持久化比例，例如 `15` → `0.15`。 */
export function parsePercentInput(value: string): string {
  if (!DECIMAL_TEXT_PATTERN.test(value)) return value;
  try {
    return numeric.divide(value, '100').serialize() ?? value;
  } catch {
    // 输入阶段允许暂时非法，由既有字段校验器在失焦/提交时提示。
    return value;
  }
}
