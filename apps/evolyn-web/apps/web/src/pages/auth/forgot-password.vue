<script setup lang="ts">
// 忘记密码页（占位）：后端重置密码接口未开放，先提供说明与返回入口
import { useRouter } from 'vue-router'

const router = useRouter()

function goLogin() {
  router.replace('/auth/login')
}
</script>

<template>
  <AuthLayout title="找回密码" subtitle="验证身份后重置登录密码">
    <el-result class="forgot-page__result" icon="info" title="找回密码功能即将上线">
      <template #sub>
        <p class="forgot-page__tip">
          上线前可联系团队管理员重置密码，<br>
          或先使用密码方式登录后在账号设置中修改。
        </p>
      </template>
      <template #extra>
        <el-button type="primary" size="large" @click="goLogin">返回登录</el-button>
      </template>
    </el-result>
  </AuthLayout>
</template>

<style lang="scss" scoped>
.forgot-page__result {
  padding: 16px 0 8px;
}

.forgot-page__tip {
  margin: 0;
  font-size: var(--el-font-size-base);
  line-height: 1.8;
  color: var(--el-text-color-secondary);
}
</style>
