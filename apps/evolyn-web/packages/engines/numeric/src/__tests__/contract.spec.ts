import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

import { numeric, NumericError, RoundingMode, type NumericInput } from '../index'

/**
 * 跨端契约测试（设计 §52/§53）：与 Go 侧 internal/platform/numeric/
 * contract_test.go 共读 docs/contracts/numeric-test-vectors.json，
 * 保证 JS Runtime == Go Runtime。
 */
interface Vector {
  operation: string
  args: (string | null)[]
  roundScale?: number
  roundMode?: keyof typeof RoundingMode
  expected?: string | null
  expectError?: string
}

const vectorsFile = join(
  fileURLToPath(import.meta.url),
  // 本文件位于 apps/evolyn-web/packages/engines/numeric/src/__tests__/
  // 首个 '..' 抵消文件名，其后 7 级上溯到仓库根
  '..',
  '..',
  '..',
  '..',
  '..',
  '..',
  '..',
  '..',
  'docs',
  'contracts',
  'numeric-test-vectors.json',
)

const doc = JSON.parse(readFileSync(vectorsFile, 'utf-8')) as { vectors: Vector[] }
expect(doc.vectors.length).toBeGreaterThan(0)

function arg(input: string | null): NumericInput {
  return input
}

function applyRound(value: ReturnType<typeof numeric.of>, v: Vector) {
  if (v.roundScale === undefined) {
    return value
  }
  const mode = v.roundMode ? RoundingMode[v.roundMode] : undefined
  return value.round(v.roundScale, mode)
}

/** 按向量执行一次运行时调用，返回可比较的结果（串）或抛 NumericError */
function run(v: Vector): string | null {
  const a = arg(v.args[0] ?? null)
  const b = arg(v.args[1] ?? null)
  const all = v.args.map((x) => arg(x ?? null))

  switch (v.operation) {
    case 'parse':
    case 'serialize':
      return numeric.of(a).serialize()
    case 'add':
      return numeric.add(a, b).serialize()
    case 'subtract':
      return numeric.subtract(a, b).serialize()
    case 'multiply':
      return numeric.multiply(a, b).serialize()
    case 'divide':
      return applyRound(numeric.divide(a, b), v).serialize()
    case 'mod':
      return numeric.mod(a, b).serialize()
    case 'abs':
      return numeric.abs(a).serialize()
    case 'negate':
      return numeric.negate(a).serialize()
    case 'sqrt':
      return applyRound(numeric.sqrt(a), v).serialize()
    case 'round':
      return numeric
        .round(a, v.roundScale ?? 0, v.roundMode ? RoundingMode[v.roundMode] : undefined)
        .serialize()
    case 'ceil':
      return numeric.ceil(a).serialize()
    case 'floor':
      return numeric.floor(a).serialize()
    case 'compare':
      return String(numeric.compare(a, b))
    case 'sum':
      return numeric.sum(all).serialize()
    case 'average':
      return applyRound(numeric.average(all), v).serialize()
    case 'min':
      return numeric.min(all)?.serialize() ?? null
    case 'max':
      return numeric.max(all)?.serialize() ?? null
    case 'product':
      return numeric.product(all).serialize()
    default:
      throw new Error(`未知契约操作: ${v.operation}`)
  }
}

describe('跨端数值契约（与 Go 共读 docs/contracts/numeric-test-vectors.json）', () => {
  for (const [index, v] of doc.vectors.entries()) {
    it(`#${index} ${v.operation}(${v.args.join(', ')})`, () => {
      if (v.expectError) {
        try {
          run(v)
        } catch (err) {
          expect(err).toBeInstanceOf(NumericError)
          expect((err as NumericError).code).toBe(v.expectError)
          return
        }
        throw new Error(`期望稳定错误码 ${v.expectError}，但未抛出`)
      }
      expect(run(v)).toBe(v.expected ?? null)
    })
  }
})
