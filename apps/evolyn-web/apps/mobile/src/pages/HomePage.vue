<script setup lang="ts">
import { showToast, Icon as VanIcon } from 'vant';
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { listApps } from '~/api/apps';
import ProfileDrawer from '~/components/ProfileDrawer.vue';

interface WorkItem {
  label: string;
  icon: string;
  badge?: number;
}

interface AppItem {
  code: string;
  label: string;
  short: string;
  tone: string;
  status?: string;
}

const drawerVisible = ref(false);
const bannerVisible = ref(true);
const keyword = ref('');
const router = useRouter();

const workItems: WorkItem[] = [
  { label: '我的待办', icon: 'bell', badge: 1 },
  { label: '我发起的', icon: 'play-circle' },
  { label: '我处理的', icon: 'checked' },
  { label: '抄送我的', icon: 'guide-o' },
];

const recentItems: WorkItem[] = [
  { label: '员工档案', icon: 'orders-o' },
  { label: '客户管理', icon: 'manager-o' },
  { label: '🌙使用说明', icon: 'orders-o' },
  { label: '跟进记录', icon: 'notes-o' },
  { label: '联系人', icon: 'calendar-o' },
  { label: '⭐项目总览', icon: 'orders-o' },
];

// 接口不可用时保留截图级演示入口，联调环境成功返回后立即替换为真实应用。
const appItems = ref<AppItem[]>([
  { code: 'demo-simple', label: '灵衍云示例应用', short: '▦', tone: 'var(--van-primary-color)' },
  { code: 'demo-inventory', label: '新进销存-标准版', short: '✪', tone: '#5799f4' },
  { code: 'demo-crm', label: 'CRM', short: 'CRM', tone: '#ef7777', status: '停用' },
  { code: 'demo-production', label: '生产小工单', short: 'ERP', tone: '#3ac0cf' },
  { code: 'demo-project', label: '项目管理', short: 'PM', tone: 'var(--van-primary-color)' },
]);

onMounted(async () => {
  try {
    const page = await listApps();
    appItems.value = page.items.map((app) => ({
      code: app.code,
      label: app.name,
      short: app.name.slice(0, 2).toUpperCase(),
      tone: 'var(--van-primary-color)',
      status: app.status === 'archived' ? '停用' : undefined,
    }));
  } catch {
    // 工作台仍可通过演示应用验收完整移动交互；真实请求恢复后刷新即可重载。
  }
});

function openFeature(label: string): void {
  showToast(`${label}正在建设中`);
}

function submitSearch(): void {
  const value = keyword.value.trim();
  showToast(value ? `正在搜索“${value}”` : '请输入搜索内容');
}

function openApp(app: AppItem): void {
  void router.push({
    name: 'mobile-app',
    params: { appCode: app.code },
    query: { name: app.label },
  });
}
</script>

<template>
  <main class="workbench-page">
    <header class="workbench-header">
      <button class="workbench-avatar" type="button" aria-label="打开个人菜单" @click="drawerVisible = true">
        @
      </button>
      <button class="workbench-switch" type="button" @click="openFeature('工作台切换')">
        <strong>工作台</strong>
        <VanIcon name="exchange" />
      </button>
      <button class="workbench-notification" type="button" aria-label="通知" @click="openFeature('通知中心')">
        <VanIcon name="bell" />
        <i aria-hidden="true" />
      </button>
    </header>

    <section v-if="bannerVisible" class="product-banner">
      <VanIcon name="video-o" />
      <button type="button" @click="openFeature('产品介绍')">
        两分钟了解本产品
      </button>
      <button type="button" aria-label="关闭介绍横幅" @click="bannerVisible = false">
        <VanIcon name="cross" />
      </button>
    </section>

    <form class="workbench-search" @submit.prevent="submitSearch">
      <VanIcon name="search" />
      <input v-model="keyword" type="search" placeholder="搜索应用、表单、仪表盘" aria-label="搜索工作台">
      <button type="button" aria-label="扫一扫" @click="openFeature('扫一扫')">
        <VanIcon name="scan" />
      </button>
    </form>

    <section class="workbench-card workbench-shortcuts" aria-label="流程快捷入口">
      <button v-for="item in workItems" :key="item.label" type="button" @click="openFeature(item.label)">
        <span class="shortcut-icon">
          <VanIcon :name="item.icon" color="var(--van-primary-color)" />
          <i v-if="item.badge">{{ item.badge }}</i>
        </span>
        <span>{{ item.label }}</span>
      </button>
      <i class="workbench-shortcuts__handle" aria-hidden="true" />
    </section>

    <section class="workbench-card recent-card">
      <h2>最近使用</h2>
      <div class="recent-grid">
        <button v-for="item in recentItems" :key="item.label" type="button" @click="openFeature(item.label)">
          <VanIcon :name="item.icon" />
          <span>{{ item.label }}</span>
        </button>
      </div>
    </section>

    <section class="workbench-card apps-card">
      <header>
        <h2>我的应用</h2>
        <button type="button" @click="openFeature('全部应用')">
          查看我的应用 <VanIcon name="arrow" />
        </button>
      </header>
      <div class="apps-grid">
        <button v-for="app in appItems" :key="app.code" type="button" @click="openApp(app)">
          <span class="app-icon" :style="{ backgroundColor: app.tone }">
            {{ app.short }}
            <i v-if="app.status">{{ app.status }}</i>
          </span>
          <span>{{ app.label }}</span>
        </button>
      </div>
    </section>

    <ProfileDrawer v-model="drawerVisible" />
  </main>
</template>

<style scoped>
.workbench-page {
  box-sizing: border-box;
  width: min(100%, 604px);
  min-height: 100dvh;
  padding: calc(16px + env(safe-area-inset-top)) 15px 34px;
  margin: 0 auto;
  color: #192437;
  background: #f5f6f8;
}

.workbench-header {
  display: grid;
  min-height: 90px;
  grid-template-columns: 64px 1fr 64px;
  align-items: center;
}

.workbench-header button {
  padding: 0;
  background: none;
  border: 0;
}

.workbench-avatar {
  display: grid;
  width: 58px;
  height: 58px;
  font-size: 27px;
  font-weight: 500;
  color: #fff;
  background: #f45156 !important;
  border-radius: 50% !important;
  place-items: center;
}

.workbench-switch {
  display: flex;
  gap: 10px;
  color: #1d293b;
  align-items: center;
  justify-content: center;
}

.workbench-switch strong {
  font-size: 25px;
  font-weight: 600;
}

.workbench-switch .van-icon {
  font-size: 25px;
  color: #576273;
}

.workbench-notification {
  position: relative;
  display: grid;
  width: 50px;
  height: 50px;
  margin-left: auto;
  font-size: 22px;
  color: #596473;
  background: #fff !important;
  border: 1px solid #dfe3e8 !important;
  border-radius: 8px !important;
  place-items: center;
}

.workbench-notification i {
  position: absolute;
  top: 8px;
  right: 9px;
  width: 8px;
  height: 8px;
  background: #ee4d55;
  border: 1px solid #fff;
  border-radius: 50%;
}

.product-banner {
  position: relative;
  display: flex;
  height: 68px;
  overflow: hidden;
  color: #fff;
  background:
    radial-gradient(ellipse at 12% 110%, rgb(255 255 255 / 14%) 0 36%, transparent 37%),
    radial-gradient(ellipse at 65% 130%, rgb(255 255 255 / 12%) 0 45%, transparent 46%),
    var(--van-primary-color);
  border-radius: 8px;
  align-items: center;
  justify-content: center;
}

.product-banner > .van-icon {
  margin-right: 15px;
  font-size: 22px;
}

.product-banner button {
  font-size: 19px;
  font-weight: 600;
  color: inherit;
  background: none;
  border: 0;
}

.product-banner button:last-child {
  position: absolute;
  right: 12px;
  padding: 8px;
  font-size: 22px;
  font-weight: 400;
  opacity: 0.58;
}

.workbench-search {
  display: flex;
  height: 54px;
  margin: 24px 0;
  padding-left: 20px;
  font-size: 21px;
  color: #8c95a2;
  background: #fff;
  border-radius: 28px;
  align-items: center;
}

.workbench-search input {
  min-width: 0;
  height: 100%;
  margin-left: 14px;
  font-size: 20px;
  color: #283447;
  background: transparent;
  border: 0;
  outline: 0;
  flex: 1;
}

.workbench-search input::placeholder {
  color: #adb3bc;
}

.workbench-search button {
  display: grid;
  width: 58px;
  height: 58px;
  padding: 0;
  margin-right: -2px;
  font-size: 25px;
  color: #596473;
  background: #fff;
  border: 0;
  border-radius: 50%;
  box-shadow: 0 3px 12px rgb(19 33 56 / 7%);
  place-items: center;
}

.workbench-card {
  background: #fff;
  border-radius: 8px;
}

.workbench-shortcuts {
  position: relative;
  display: grid;
  min-height: 121px;
  grid-template-columns: repeat(4, 1fr);
}

.workbench-shortcuts button {
  display: flex;
  padding: 19px 4px 15px;
  gap: 13px;
  font-size: 17px;
  color: #273247;
  background: none;
  border: 0;
  flex-direction: column;
  align-items: center;
}

.shortcut-icon {
  position: relative;
  display: inline-block;
  font-size: 37px;
  line-height: 1;
  color: var(--van-primary-color);
}

.shortcut-icon .van-icon {
  color: var(--van-primary-color);
}

.shortcut-icon i {
  position: absolute;
  top: -11px;
  right: -16px;
  display: grid;
  width: 27px;
  height: 27px;
  font-size: 15px;
  font-style: normal;
  color: #fff;
  background: #ec4c52;
  border: 2px solid #fff;
  border-radius: 50%;
  place-items: center;
}

.workbench-shortcuts__handle {
  position: absolute;
  bottom: 11px;
  left: calc(50% - 10px);
  width: 20px;
  height: 4px;
  background: #bac0c8;
  border-radius: 4px;
}

.recent-card,
.apps-card {
  padding: 15px 22px 19px;
  margin-top: 12px;
}

.workbench-card h2 {
  margin: 0 0 16px;
  font-size: 22px;
  font-weight: 600;
}

.recent-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(2, 1fr);
}

.recent-grid button {
  display: flex;
  min-width: 0;
  height: 63px;
  padding: 0 20px;
  gap: 18px;
  font-size: 18px;
  color: #273247;
  text-align: left;
  background: #f5f6f8;
  border: 0;
  border-radius: 7px;
  align-items: center;
}

.recent-grid .van-icon {
  font-size: 30px;
  color: var(--van-primary-color);
  flex: 0 0 auto;
}

.recent-grid span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.apps-card header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.apps-card header button {
  padding: 0;
  font-size: 17px;
  color: #687280;
  background: none;
  border: 0;
}

.apps-grid {
  display: grid;
  row-gap: 23px;
  grid-template-columns: repeat(4, 1fr);
}

.apps-grid button {
  display: flex;
  min-width: 0;
  padding: 0 5px;
  gap: 9px;
  font-size: 15px;
  line-height: 1.25;
  color: #263145;
  background: none;
  border: 0;
  flex-direction: column;
  align-items: center;
}

.app-icon {
  position: relative;
  display: grid;
  width: 62px;
  height: 62px;
  font-size: 14px;
  font-weight: 700;
  color: #fff;
  border-radius: 11px;
  place-items: center;
}

.app-icon i {
  position: absolute;
  top: -12px;
  right: -15px;
  padding: 4px 8px;
  font-size: 12px;
  font-style: normal;
  font-weight: 400;
  background: #ee4d55;
  border-radius: 12px;
}

@media (max-width: 420px) {
  .workbench-page {
    padding-right: 12px;
    padding-left: 12px;
  }

  .workbench-header {
    min-height: 72px;
  }

  .workbench-avatar {
    width: 48px;
    height: 48px;
    font-size: 23px;
  }

  .workbench-switch strong {
    font-size: 21px;
  }

  .product-banner {
    height: 56px;
  }

  .product-banner button {
    font-size: 16px;
  }

  .workbench-search {
    height: 48px;
    margin: 18px 0;
  }

  .workbench-search input {
    font-size: 16px;
  }

  .workbench-shortcuts button {
    font-size: 14px;
  }

  .recent-card,
  .apps-card {
    padding-right: 14px;
    padding-left: 14px;
  }

  .recent-grid button {
    padding: 0 12px;
    gap: 10px;
    font-size: 15px;
  }
}
</style>
