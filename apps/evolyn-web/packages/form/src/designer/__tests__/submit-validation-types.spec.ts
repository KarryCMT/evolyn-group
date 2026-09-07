import { describe, expect, it } from 'vitest';
import { reactive } from 'vue';
import { cloneSubmitValidatorDraft } from '../submit-validation-types';

describe('提交校验设计草稿', () => {
  it('将响应式规则投影为可独立编辑的纯协议对象', () => {
    const source = reactive({
      formula: 'LEN($_widget_phone#) == 11',
      remind: '联系电话长度不正确',
      remark: '联系电话',
      realtime: true,
      failAction: 0 as const,
    });

    const draft = cloneSubmitValidatorDraft(source);
    draft.remind = '新的提示';

    expect(draft).toEqual({
      formula: 'LEN($_widget_phone#) == 11',
      remind: '新的提示',
      remark: '联系电话',
      realtime: true,
      failAction: 0,
    });
    expect(source.remind).toBe('联系电话长度不正确');
  });
});
