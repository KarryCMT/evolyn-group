<script setup lang="ts">
import { showConfirmDialog, showToast, Icon as VanIcon, NavBar as VanNavBar } from 'vant';
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '~/stores/auth';

const router = useRouter();
const auth = useAuthStore();

const account = computed(() => auth.userInfo?.account);
const displayName = computed(() => auth.displayName || '灵衍云用户');
const userId = computed(() => auth.userInfo?.member.memberCode || String(account.value?.id ?? '--'));
const locale = computed(() => auth.userInfo?.tenant.config.locale || '简体中文');

function openSetting(label: string): void {
  showToast(`${label}请前往电脑端完成`);
}

async function confirmCancellation(): Promise<void> {
  try {
    await showConfirmDialog({
      title: '注销账号',
      message: '账号注销会影响你加入的全部团队。为确保数据安全，请前往电脑端完成身份验证。',
      confirmButtonText: '我知道了',
      cancelButtonText: '取消',
      confirmButtonColor: '#ee4d55',
    });
  } catch {
    // 用户取消时保持当前页面，不产生任何账号写操作。
  }
}
</script>

<template>
  <main class="settings-page">
    <VanNavBar
      class="settings-nav"
      title="个人设置"
      left-arrow
      safe-area-inset-top
      @click-left="router.back()"
    />

    <div class="settings-content">
      <section class="settings-card">
        <button type="button" @click="openSetting('通讯录头像')">
          <span>通讯录头像</span>
          <span class="settings-avatar">
            <img v-if="account?.avatar" :src="account.avatar" alt="">
            <i v-else>@</i>
          </span>
        </button>
        <button type="button" @click="openSetting('通讯录姓名')">
          <span>通讯录姓名</span>
          <small>{{ displayName }}</small>
          <VanIcon name="arrow" />
        </button>
        <div class="settings-row">
          <span>用户 ID</span>
          <small>{{ userId }}</small>
        </div>
        <button type="button" @click="openSetting('语言')">
          <span>语言</span>
          <small>{{ locale }}</small>
          <VanIcon name="arrow" />
        </button>
      </section>

      <section class="settings-card">
        <button type="button" @click="openSetting('密码设置')">
          <span>密码</span>
          <small>{{ account?.passwordInitialized ? '已设置' : '未设置' }}</small>
          <VanIcon name="arrow" />
        </button>
        <button type="button" @click="openSetting('手机绑定')">
          <span>手机</span>
          <small>{{ account?.phone ? '已绑定' : '未绑定' }}</small>
          <VanIcon name="arrow" />
        </button>
        <button type="button" @click="openSetting('邮箱绑定')">
          <span>邮箱</span>
          <small>{{ account?.email ? '已绑定' : '未绑定' }}</small>
          <VanIcon name="arrow" />
        </button>
      </section>

      <section class="settings-card settings-card--danger">
        <button type="button" @click="confirmCancellation">
          注销账号
        </button>
      </section>
    </div>
  </main>
</template>

<style scoped>
.settings-page {
  width: min(100%, 604px);
  min-height: 100dvh;
  margin: 0 auto;
  color: #1f2a3d;
  background: #f5f6f8;
}

.settings-nav {
  --van-nav-bar-background: #f5f6f8;
  --van-nav-bar-title-text-color: #1f2a3d;
  --van-nav-bar-icon-color: #4e5969;
  --van-nav-bar-height: 58px;
}

.settings-nav::after {
  display: none;
}

.settings-content {
  display: grid;
  padding: 28px 15px 60px;
  gap: 30px;
}

.settings-card {
  overflow: hidden;
  background: #fff;
  border-radius: 12px;
}

.settings-card > button,
.settings-row {
  position: relative;
  box-sizing: border-box;
  display: flex;
  width: 100%;
  min-height: 85px;
  padding: 0 24px;
  font-size: 20px;
  color: #1f2a3d;
  text-align: left;
  background: none;
  border: 0;
  align-items: center;
}

.settings-card > :not(:last-child)::after {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 20px;
  height: 1px;
  content: '';
  background: #e7e9ed;
}

.settings-card > button > span:first-child,
.settings-row > span:first-child {
  flex: 1;
}

.settings-card small,
.settings-row small {
  max-width: 66%;
  overflow: hidden;
  font-size: 18px;
  color: #8b929d;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-card .van-icon {
  margin-left: 8px;
  font-size: 21px;
  color: #8d95a0;
}

.settings-avatar {
  display: grid;
  width: 51px;
  height: 51px;
  overflow: hidden;
  color: #fff;
  background: #f45156;
  border-radius: 50%;
  place-items: center;
}

.settings-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.settings-avatar i {
  font-size: 23px;
  font-style: normal;
}

.settings-card--danger button {
  min-height: 86px;
  color: #ee4d55;
  justify-content: center;
}

@media (max-width: 420px) {
  .settings-content {
    padding-top: 20px;
    gap: 20px;
  }

  .settings-card > button,
  .settings-row {
    min-height: 68px;
    padding: 0 20px;
    font-size: 17px;
  }

  .settings-card small,
  .settings-row small {
    font-size: 16px;
  }
}
</style>
