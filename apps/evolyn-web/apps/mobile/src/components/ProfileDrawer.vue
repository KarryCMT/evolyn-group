<script setup lang="ts">
import {
  showConfirmDialog,
  showToast,
  Icon as VanIcon,
  Popup as VanPopup,
} from 'vant';
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '~/stores/auth';

const visible = defineModel<boolean>({ required: true });
const router = useRouter();
const auth = useAuthStore();

const profileName = computed(() => auth.displayName || '灵衍云用户');
const roleLabel = computed(() => auth.isTenantOwner ? '创建者' : '成员');
const tenantName = computed(() => auth.userInfo?.tenant.name || '当前团队');

async function openSettings(): Promise<void> {
  visible.value = false;
  await router.push('/account/settings');
}

async function handleLogout(): Promise<void> {
  try {
    await showConfirmDialog({
      title: '退出登录',
      message: '确定要退出当前账号吗？',
      confirmButtonText: '退出',
      confirmButtonColor: '#ee4d55',
    });
  } catch {
    return;
  }
  await auth.logout();
  visible.value = false;
  await router.replace('/auth/login');
}

function unavailable(label: string): void {
  showToast(`${label}正在建设中`);
}
</script>

<template>
  <VanPopup
    v-model:show="visible"
    class="profile-drawer"
    position="left"
    :style="{ width: '80%', height: '100%' }"
    teleport="body"
  >
    <aside class="profile-drawer__content" aria-label="个人菜单">
      <header class="profile-drawer__profile">
        <div class="profile-avatar">
          @
        </div>
        <div class="profile-copy">
          <strong>{{ profileName }}</strong>
          <div>
            <span>{{ roleLabel }}</span>
            <span class="profile-plan">免费版</span>
          </div>
        </div>
        <button class="profile-drawer__scan" type="button" aria-label="扫一扫" @click="unavailable('扫一扫')">
          <VanIcon name="scan" />
        </button>
      </header>

      <nav class="profile-drawer__menus">
        <section class="drawer-card">
          <button type="button" @click="unavailable('切换企业/团队')">
            <VanIcon name="friends-o" />
            <span>切换企业/团队</span>
            <small>{{ tenantName }}</small>
            <VanIcon class="drawer-card__arrow" name="arrow" />
          </button>
          <button type="button" @click="unavailable('通讯录')">
            <VanIcon name="contact-o" />
            <span>通讯录</span>
            <VanIcon class="drawer-card__arrow" name="arrow" />
          </button>
          <button type="button" @click="openSettings">
            <VanIcon name="user-o" />
            <span>个人设置</span>
            <VanIcon class="drawer-card__arrow" name="arrow" />
          </button>
          <button type="button" @click="unavailable('我的收藏')">
            <VanIcon name="star-o" />
            <span>我的收藏</span>
            <VanIcon class="drawer-card__arrow" name="arrow" />
          </button>
        </section>

        <section class="drawer-card drawer-card--single">
          <button type="button" @click="unavailable('模板中心')">
            <VanIcon name="gem-o" />
            <span>模板中心</span>
            <VanIcon class="drawer-card__arrow" name="arrow" />
          </button>
        </section>

        <section class="drawer-card drawer-card--help">
          <div class="drawer-card__help-title">
            <VanIcon name="question-o" />
            <span>帮助中心</span>
            <VanIcon class="drawer-card__arrow" name="arrow" />
          </div>
          <div class="help-actions">
            <button type="button" @click="unavailable('帮助文档')">
              <span class="help-actions__icon help-actions__icon--violet"><VanIcon name="records-o" /></span>
              <span>帮助文档</span>
            </button>
            <button type="button" @click="unavailable('视频教程')">
              <span class="help-actions__icon help-actions__icon--cyan"><VanIcon name="video-o" /></span>
              <span>视频教程</span>
            </button>
            <button type="button" @click="unavailable('技术支持')">
              <span class="help-actions__icon"><VanIcon name="chat-o" /></span>
              <span>技术支持</span>
            </button>
          </div>
        </section>

        <section class="drawer-card drawer-card--single">
          <button type="button" @click="unavailable('需求反馈')">
            <VanIcon name="comment-o" />
            <span>需求反馈</span>
            <VanIcon class="drawer-card__arrow" name="arrow" />
          </button>
        </section>

        <section class="drawer-card drawer-card--logout">
          <button type="button" @click="handleLogout">
            退出登录
          </button>
        </section>
      </nav>
    </aside>
  </VanPopup>
</template>

<style scoped>
.profile-drawer {
  max-width: 482px;
  background: #f5f6f8;
}

.profile-drawer__content {
  box-sizing: border-box;
  height: 100%;
  padding: calc(28px + env(safe-area-inset-top)) 16px 24px;
  overflow-y: auto;
}

.profile-drawer__profile {
  display: flex;
  min-height: 104px;
  padding: 0 13px;
  align-items: center;
}

.profile-avatar {
  display: grid;
  width: 58px;
  height: 58px;
  font-size: 28px;
  font-weight: 500;
  color: #fff;
  background: #f45156;
  border-radius: 50%;
  place-items: center;
  flex: 0 0 auto;
}

.profile-copy {
  min-width: 0;
  margin-left: 16px;
  color: #1f293b;
  flex: 1;
}

.profile-copy strong {
  display: block;
  overflow: hidden;
  font-size: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-copy div {
  display: flex;
  margin-top: 4px;
  gap: 10px;
  font-size: 15px;
  color: #8b93a0;
  align-items: center;
}

.profile-plan {
  padding: 2px 8px;
  color: var(--van-primary-color);
  border: 1px solid color-mix(in srgb, var(--van-primary-color) 45%, white);
  border-radius: 10px;
}

.profile-drawer__scan {
  padding: 8px;
  font-size: 30px;
  color: #596473;
  background: transparent;
  border: 0;
}

.profile-drawer__menus {
  display: grid;
  gap: 14px;
}

.drawer-card {
  overflow: hidden;
  background: #fff;
  border-radius: 12px;
}

.drawer-card > button,
.drawer-card__help-title {
  box-sizing: border-box;
  display: flex;
  width: 100%;
  min-height: 74px;
  padding: 0 25px;
  gap: 20px;
  font-size: 20px;
  color: #202b3d;
  text-align: left;
  background: transparent;
  border: 0;
  align-items: center;
}

.drawer-card > button > .van-icon,
.drawer-card__help-title > .van-icon {
  font-size: 25px;
  color: #657080;
}

.drawer-card > button span,
.drawer-card__help-title span {
  flex: 1;
}

.drawer-card > button small {
  max-width: 110px;
  overflow: hidden;
  font-size: 15px;
  color: #8b93a0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.drawer-card .drawer-card__arrow {
  font-size: 18px;
  color: #b1b7c0;
}

.drawer-card--single > button {
  min-height: 78px;
}

.drawer-card__help-title {
  min-height: 68px;
}

.help-actions {
  display: grid;
  padding: 4px 17px 28px;
  grid-template-columns: repeat(3, 1fr);
}

.help-actions button {
  display: flex;
  min-width: 0;
  padding: 0;
  gap: 9px;
  font-size: 15px;
  color: #202b3d;
  background: none;
  border: 0;
  flex-direction: column;
  align-items: center;
}

.help-actions__icon {
  display: grid;
  width: 48px;
  height: 48px;
  font-size: 23px;
  color: var(--van-primary-color);
  border: 2px solid color-mix(in srgb, var(--van-primary-color) 72%, white);
  border-radius: 50%;
  place-items: center;
}

.help-actions__icon--violet {
  color: #635bff;
  border-color: #8a83ff;
}

.help-actions__icon--cyan {
  color: #24bdd4;
  border-color: #67d4e3;
}

.drawer-card--logout button {
  min-height: 64px;
  color: #ee4d55;
  justify-content: center;
}
</style>
