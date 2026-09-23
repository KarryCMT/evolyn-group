import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

import type { LabelSchema } from '../schema/label-schema.js';
import { validateLabelSchema, type LabelValidationIssue } from '../schema/validator.js';

interface LabelContractVector {
  name: string;
  schema: LabelSchema;
  expectedIssues: LabelValidationIssue[];
}

const vectorsFile = join(
  fileURLToPath(import.meta.url),
  // 本文件位于 apps/evolyn-web/packages/engines/label/src/__tests__/，上溯至仓库根。
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
  'label-schema-test-vectors.json',
);

const document = JSON.parse(readFileSync(vectorsFile, 'utf-8')) as {
  vectors: LabelContractVector[];
};

describe('LabelSchema 跨端校验契约', () => {
  for (const vector of document.vectors) {
    it(vector.name, () => {
      expect(validateLabelSchema(vector.schema)).toEqual(vector.expectedIssues);
    });
  }
});
