<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn';
</script>

<template>
  <!-- Element Plus 内置日期、时间、分页等控件统一跟随产品默认简体中文。 -->
  <el-config-provider :locale="zhCn">
    <RouterView />
  </el-config-provider>
</template>

<style></style>
