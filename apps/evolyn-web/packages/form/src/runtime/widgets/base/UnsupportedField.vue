<script setup lang="ts">
import { computed } from 'vue';
import type { RuntimeFieldProps } from '../../types';

/**
 * 未支持字段占位：未知或当前阶段未开放的控件类型不可静默丢弃，渲染受控的
 * 「此字段暂不支持」状态；FormFieldHost 会把诊断回传给宿主，发布侧服务端
 * 白名单应尽量在发布校验阶段阻止该情况进入已发布版本。
 */
const props = defineProps<RuntimeFieldProps>();

const hasLabel = computed(() => props.item.label.trim() !== '');
</script>

<template>
  <div class="evf-unsupported" role="note">
    <span v-if="hasLabel" class="evf-unsupported__label">{{ item.label }}：</span>
    <span>此字段类型（{{ item.widget.type }}）暂不支持填写</span>
  </div>
</template>
