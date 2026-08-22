<script setup lang="ts">
// 第三方登录入口（GitHub / 微信）：后端 OAuth 能力已就绪，但授权页跳转
// 需要下发 client_id 的配置接口；当前以提示占位，接入后改为跳转各提供方授权页
import { ElMessage } from 'element-plus';
import { RiGithubFill, RiWechatFill } from '@remixicon/vue';

const providers = [
  // 品牌入口使用对应的 Fill 图标，方便用户快速识别登录方式。
  { key: 'github', label: 'GitHub', icon: RiGithubFill },
  { key: 'wechat', label: '微信', icon: RiWechatFill },
] as const;

function handleSelect(label: string) {
  ElMessage.info(`「${label}」登录待 OAuth 配置下发后开放`);
}
</script>

<template>
  <div class="oauth-login-panel">
    <el-divider class="oauth-login-panel__divider">其他登录方式</el-divider>
    <div class="oauth-login-panel__list">
      <el-button
        v-for="provider in providers"
        :key="provider.key"
        class="oauth-login-panel__item"
        :icon="provider.icon"
        plain
        size="large"
        @click="handleSelect(provider.label)"
      >
        {{ provider.label }}
      </el-button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.oauth-login-panel {
  margin-top: 24px;
}

.oauth-login-panel__divider {
  margin: 0 0 20px;
}

.oauth-login-panel__list {
  display: flex;
  gap: 12px;
}

.oauth-login-panel__item {
  flex: 1;
}
</style>
