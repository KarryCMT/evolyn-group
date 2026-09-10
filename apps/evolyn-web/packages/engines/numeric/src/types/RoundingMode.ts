/**
 * RoundingMode 平台自有舍入枚举（设计 §11）。
 *
 * 业务层只允许使用本枚举；Decimal.ROUND_* 常量由适配层内部映射，
 * 禁止业务包直接引用 decimal.js 的舍入常量。
 */
export enum RoundingMode {
  /** 远离零方向舍入 */
  UP = 'UP',
  /** 向零方向舍入（截断） */
  DOWN = 'DOWN',
  /** 向正无穷方向舍入 */
  CEIL = 'CEIL',
  /** 向负无穷方向舍入 */
  FLOOR = 'FLOOR',
  /** 四舍五入（半值远离零），平台默认 */
  HALF_UP = 'HALF_UP',
  /** 半值向零舍入 */
  HALF_DOWN = 'HALF_DOWN',
  /** 银行家舍入（半值取偶），财务对账场景 */
  HALF_EVEN = 'HALF_EVEN',
}
