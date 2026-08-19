<script setup lang="ts">
// 顶栏语言切换（占位）：i18n 方案落地前仅记录选择并提示，
// 落地后在此接入 vue-i18n 的 locale 切换
import { computed, shallowRef } from 'vue'
import { ArrowDown } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

/** 本地语言偏好存储键 */
const LOCALE_KEY = 'evolyn.locale'

const locales = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en', label: 'English' },
] as const

const current = shallowRef(localStorage.getItem(LOCALE_KEY) ?? 'zh-CN')
const currentLabel = computed(
  () => locales.find(locale => locale.value === current.value)?.label ?? '简体中文',
)

function handleChange(value: string | number | object) {
  current.value = String(value)
  localStorage.setItem(LOCALE_KEY, current.value)
  ElMessage.info('多语言支持即将上线')
}
</script>

<template>
  <el-dropdown class="locale-switch" trigger="click" @command="handleChange">
    <span class="locale-switch__trigger" role="button" tabindex="0">
      {{ currentLabel }}
      <el-icon class="locale-switch__arrow"><ArrowDown /></el-icon>
    </span>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="locale in locales"
          :key="locale.value"
          :command="locale.value"
          :disabled="locale.value === current"
        >
          {{ locale.label }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<style lang="scss" scoped>
.locale-switch__trigger {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-regular);
  cursor: pointer;
  outline: none;

  &:hover {
    color: var(--el-color-primary);
  }
}

.locale-switch__arrow {
  font-size: 12px;
}
</style>
