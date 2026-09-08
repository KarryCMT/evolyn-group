import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { evaluateSubmitValidators } from '../submit-validation';

type GoldenFixture = {
  name: string;
  formula: string;
  value: unknown;
  passed: boolean;
  remind: string;
};

const fixtures = JSON.parse(
  readFileSync(
    resolve(
      __dirname,
      '../../../../../../evolyn-core/internal/platform/form/service/testdata/submit_validation_golden.json',
    ),
    'utf8',
  ),
) as GoldenFixture[];

describe('submit-validation shared golden fixtures', () => {
  for (const fixture of fixtures) {
    it(fixture.name, () => {
      const failures = evaluateSubmitValidators(
        {
          items: [
            {
              label: '联系电话',
              description: '',
              labelHidden: false,
              lineWidth: 12,
              widget: {
                type: 'text',
                widgetName: '_widget_phone',
                visible: true,
                enable: true,
                allowBlank: true,
              },
            },
            {
              label: '选项',
              description: '',
              labelHidden: false,
              lineWidth: 12,
              widget: {
                type: 'text',
                widgetName: '_widget_option',
                visible: true,
                enable: true,
                allowBlank: true,
              },
            },
          ],
          validators: [
            {
              formula: fixture.formula,
              remind: fixture.remind,
              realtime: false,
              failAction: 0,
              remark: '',
            },
          ],
        },
        {
          values: { _widget_phone: fixture.value as never, _widget_option: fixture.value as never },
          isVisible: () => true,
        },
      );
      expect(failures.length > 0).toBe(!fixture.passed);
    });
  }
});
