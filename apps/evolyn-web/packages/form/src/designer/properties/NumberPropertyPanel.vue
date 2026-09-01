<script setup lang="ts">
import { ElInputNumber } from 'element-plus';
import type { NumberWidget } from '../../schema/types';
import FormSchemaPropertySection from './FormSchemaPropertySection.vue';
defineProps<{ widget: NumberWidget }>();
</script>
<template>
  <FormSchemaPropertySection title="默认值">
    <el-input-number
      :model-value="widget.defaultValue ?? undefined"
      placeholder="不设置"
      @update:model-value="widget.defaultValue = $event ?? null"
    />
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="数值范围">
    <div class="form-schema-property__pair">
      <div>
        <label class="form-schema-property__control-label" for="number-min">最小值</label>
        <el-input-number
          id="number-min"
          :model-value="widget.min ?? undefined"
          placeholder="不限"
          @update:model-value="widget.min = $event ?? null"
        />
      </div>
      <div>
        <label class="form-schema-property__control-label" for="number-max">最大值</label>
        <el-input-number
          id="number-max"
          :model-value="widget.max ?? undefined"
          placeholder="不限"
          @update:model-value="widget.max = $event ?? null"
        />
      </div>
    </div>
    <div>
      <label class="form-schema-property__control-label" for="number-precision">小数位数</label>
      <el-input-number
        id="number-precision"
        :model-value="widget.precision ?? undefined"
        :min="0"
        :max="8"
        placeholder="不限"
        @update:model-value="widget.precision = $event ?? null"
      />
    </div>
  </FormSchemaPropertySection>
</template>
