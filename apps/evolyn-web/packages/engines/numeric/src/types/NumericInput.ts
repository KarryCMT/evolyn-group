/**
 * NumericInput 数值运行时的统一输入类型（设计 §7.1）。
 *
 * - 平台规范是 decimal string（'123.45'），完整精度直通；
 * - number 仅作兼容入口：其可能在进入运行时前已发生二进制精度损失，
 *   由适配层做有限性校验（禁 NaN/Infinity）；
 * - null/undefined/'' 为表单空值，按 EmptyValuePolicy 处理，业务层不得散落判空。
 */
export type NumericInput = string | number | Numeric | null | undefined;

// 本地声明以避免循环导入：Numeric 在 core/Numeric.ts 定义并 re-export
import type { Numeric } from '../core/Numeric';
