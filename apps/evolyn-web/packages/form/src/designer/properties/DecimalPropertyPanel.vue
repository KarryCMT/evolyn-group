<script setup lang="ts">
import { ElInput, ElInputNumber, ElOption, ElSelect } from 'element-plus';
import {
  NUMERIC_FIELD_LIMITS,
  NUMERIC_ROUNDING_MODES,
  effectiveNumericPrecision,
  effectiveNumericScale,
} from '../../schema/numeric';
import { NUMERIC_ROUNDING_MODE_LABELS } from '../../schema/dictionary';
import type { DecimalFamilyWidget, RoundingModeValue } from '../../schema/types';
import DefaultValueModeSelect from './DefaultValueModeSelect.vue';
import FormSchemaPropertySection from './FormSchemaPropertySection.vue';

/**
 * 数值字段族（decimal/money/percent）专属面板：min/max/defaultValue 均按
 * decimal string 编辑（值协议 §15/§27，禁 float 控件承载高精度值）；
 * precision/scale 决定物理列 NUMERIC(p,s)，发布后不可变，缺省展示按类型
 * 生效的默认值；rounding 供计算链消费（Phase 4 仅入协议）。
 */
const props = defineProps<{ widget: DecimalFamilyWidget }>();

/** 文本输入回写：空串收敛 null（未启用语义），与协议缺省一致。 */
function setDecimalText(key: 'min' | 'max' | 'defaultValue', value: string): void {
  const trimmed = value.trim();
  (props.widget[key] as string | null) = trimmed === '' ? null : trimmed;
}

function textValue(value: string | null | undefined): string {
  return value ?? '';
}
</script>

<template>
  <FormSchemaPropertySection title="默认值">
    <DefaultValueModeSelect />
    <el-input
      :model-value="textValue(widget.defaultValue)"
      placeholder="不设置（十进制数字，如 123.45）"
      @update:model-value="setDecimalText('defaultValue', String($event ?? ''))"
    />
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="数值范围">
    <div class="form-schema-property__pair">
      <div>
        <label class="form-schema-property__control-label" for="decimal-min">最小值</label>
        <el-input
          id="decimal-min"
          :model-value="textValue(widget.min)"
          placeholder="不限（十进制数字）"
          @update:model-value="setDecimalText('min', String($event ?? ''))"
        />
      </div>
      <div>
        <label class="form-schema-property__control-label" for="decimal-max">最大值</label>
        <el-input
          id="decimal-max"
          :model-value="textValue(widget.max)"
          placeholder="不限（十进制数字）"
          @update:model-value="setDecimalText('max', String($event ?? ''))"
        />
      </div>
    </div>
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="精度与舍入">
    <div class="form-schema-property__pair">
      <div>
        <label class="form-schema-property__control-label" for="decimal-precision">
          有效位数
        </label>
        <el-input-number
          id="decimal-precision"
          :model-value="widget.precision ?? undefined"
          :min="NUMERIC_FIELD_LIMITS.precisionMin"
          :max="NUMERIC_FIELD_LIMITS.precisionMax"
          :placeholder="String(effectiveNumericPrecision(widget))"
          @update:model-value="widget.precision = $event ?? null"
        />
      </div>
      <div>
        <label class="form-schema-property__control-label" for="decimal-scale">小数位数</label>
        <el-input-number
          id="decimal-scale"
          :model-value="widget.scale ?? undefined"
          :min="NUMERIC_FIELD_LIMITS.scaleMin"
          :max="NUMERIC_FIELD_LIMITS.scaleMax"
          :placeholder="String(effectiveNumericScale(widget))"
          @update:model-value="widget.scale = $event ?? null"
        />
      </div>
    </div>
    <div>
      <label class="form-schema-property__control-label" for="decimal-rounding">舍入模式</label>
      <el-select
        id="decimal-rounding"
        :model-value="widget.rounding ?? undefined"
        placeholder="未设置（计算链默认 HALF_UP）"
        clearable
        @update:model-value="
          widget.rounding = ($event as RoundingModeValue | undefined) ?? undefined
        "
      >
        <el-option
          v-for="mode in NUMERIC_ROUNDING_MODES"
          :key="mode"
          :label="NUMERIC_ROUNDING_MODE_LABELS[mode]"
          :value="mode"
        />
      </el-select>
    </div>
  </FormSchemaPropertySection>
</template>
