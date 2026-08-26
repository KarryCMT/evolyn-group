<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus';
import { watch } from 'vue';
import { RiComputerFill, RiShutDownFill } from '@remixicon/vue';
import { useAccountSessions } from '~/composables/useAccountSessions';
import type { AccountSession } from '~/types';

defineOptions({ name: 'AccountSessionDrawer' });

const props = defineProps<{
  modelValue: boolean;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  sessionRevoked: [];
}>();

const { sessions, loading, revokingSID, loadSessions, revokeSession } = useAccountSessions();

const authMethodLabels: Record<string, string> = {
  password: '密码登录',
  sms: '短信验证码登录',
  oauth: '第三方登录',
  register: '注册即登录',
};

function sessionLocation(session: AccountSession) {
  return [session.location || '未知地点', session.ip || '未知 IP'].join(' · ');
}

function sessionMethod(session: AccountSession) {
  const method = authMethodLabels[session.authMethod] ?? '未知登录方式';
  return session.mfaMethod ? `${method} · 已完成二次验证` : method;
}

async function handleRevoke(session: AccountSession) {
  try {
    await ElMessageBox.confirm('下线后，该设备上的账号将立即退出登录。是否继续？', '确认下线设备', {
      confirmButtonText: '确认下线',
      cancelButtonText: '取消',
      type: 'warning',
    });
  } catch {
    return;
  }

  try {
    await revokeSession(session.sid);
    ElMessage.success('设备已下线');
    emit('sessionRevoked');
  } catch {
    ElMessage.error('设备下线失败，请稍后重试');
  }
}

// 抽屉每次打开均刷新：不同设备的新登录或已撤销状态应立即反映。
watch(
  () => props.modelValue,
  (visible) => {
    if (visible) void loadSessions();
  },
);
</script>

<template>
  <el-drawer
    :model-value="modelValue"
    class="account-session-drawer-panel"
    title="登录设备"
    direction="rtl"
    size="480px"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <section class="account-session-drawer" aria-label="活跃登录设备">
      <div class="account-session-drawer__toolbar">
        <span>下线不再使用的设备，保护账号安全。</span>
        <el-button link type="primary" :loading="loading" @click="loadSessions">刷新</el-button>
      </div>

      <el-scrollbar class="account-session-drawer__scrollbar">
        <div v-loading="loading" class="account-session-drawer__list">
          <el-empty v-if="!sessions.length && !loading" description="暂无活跃登录设备" />
          <article
            v-for="session in sessions"
            :key="session.sid"
            class="account-session-drawer__item"
          >
            <el-icon class="account-session-drawer__device"><RiComputerFill /></el-icon>
            <div class="account-session-drawer__details">
              <strong>{{ session.userAgent || '未知设备' }}</strong>
              <span>{{ sessionLocation(session) }}</span>
              <span>{{ sessionMethod(session) }}</span>
              <span>最近活跃：{{ session.lastSeenAt }}</span>
            </div>
            <el-button
              link
              type="danger"
              :loading="revokingSID === session.sid"
              @click="handleRevoke(session)"
            >
              <el-icon><RiShutDownFill /></el-icon>
              下线
            </el-button>
          </article>
        </div>
      </el-scrollbar>
    </section>
  </el-drawer>
</template>

<style scoped lang="scss">
.account-session-drawer {
  display: flex;
  height: 100%;
  flex-direction: column;

  &__toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 14px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  &__scrollbar {
    min-height: 0;
    flex: 1;
  }

  &__list {
    display: flex;
    min-height: 100%;
    flex-direction: column;
    gap: 10px;
    padding-right: 8px;
  }

  &__item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  &__device {
    flex: none;
    color: var(--el-color-primary);
    font-size: 24px;
  }

  &__details {
    display: flex;
    min-width: 0;
    flex: 1;
    flex-direction: column;
    gap: 3px;
  }

  &__details > strong,
  &__details > span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__details > strong {
    color: var(--el-text-color-primary);
    font-size: 14px;
  }

  &__details > span {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}
</style>
