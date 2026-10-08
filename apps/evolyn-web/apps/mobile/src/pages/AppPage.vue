<script setup lang="ts">
import type { MobileAppMenu, MobileMenuNode } from '~/types/app';
import { showToast, Empty as VanEmpty, Icon as VanIcon, Loading as VanLoading } from 'vant';
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getAppByCode, getAppMenuByCode } from '~/api/apps';
import MobileWorkflowCenter from '~/components/MobileWorkflowCenter.vue';
import { useAuthStore } from '~/stores/auth';

type AppTab = 'directory' | 'workflow';

interface MenuListItem {
  menuCode: string;
  name: string;
  type: MobileMenuNode['type'];
  targetCode: string | null;
  formType: 'standard' | 'workflow' | null;
  color: string;
  icon: string;
  depth: number;
}

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const activeTab = ref<AppTab>('directory');
const keyword = ref('');
const appName = ref('应用');
const menu = ref<MobileAppMenu | null>(null);
const loading = ref(true);
const errorMessage = ref('');
const favorited = ref(false);

const appCode = computed(() => {
  const value = route.params.appCode;
  return Array.isArray(value) ? (value[0] ?? '') : (value ?? '');
});
const tenantName = computed(() => auth.userInfo?.tenant.name || '灵衍云');
const isDemo = computed(() => appCode.value.startsWith('demo-'));

const menuItems = computed(() => {
  if (!menu.value) return [];
  const result: MenuListItem[] = [];
  const visit = (menuCode: string, depth: number): void => {
    const node = menu.value?.nodeMap[menuCode];
    if (!node || !node.capabilities.view) return;
    result.push({
      menuCode: node.menuId,
      name: node.name,
      type: node.type,
      targetCode: node.target?.code ?? null,
      formType: node.target?.type === 'form' ? node.target.formType : null,
      color: normalizeMenuColor(node.color, node.type),
      icon: menuIcon(node.type),
      depth,
    });
    Object.values(menu.value?.nodeMap ?? {})
      .filter((child) => child.parentMenuId === node.menuId)
      .sort(compareMenuNodes)
      .forEach((child) => visit(child.menuId, depth + 1));
  };
  menu.value.rootMenuIds.forEach((code) => visit(code, 0));
  const query = keyword.value.trim().toLocaleLowerCase();
  return query ? result.filter((item) => item.name.toLocaleLowerCase().includes(query)) : result;
});

function compareMenuNodes(left: MobileMenuNode, right: MobileMenuNode): number {
  return left.sortOrder === right.sortOrder
    ? left.menuId.localeCompare(right.menuId)
    : left.sortOrder - right.sortOrder;
}

function menuIcon(type: MobileMenuNode['type']): string {
  if (type === 'dashboard') return 'bar-chart-o';
  if (type === 'page') return 'notes-o';
  if (type === 'group') return 'cluster-o';
  return 'orders-o';
}

function normalizeMenuColor(color: string | null, type: MobileMenuNode['type']): string {
  if (color && !/green|#0[89a-f]|#1[0-9a-f]|#2[0-9a-f]/i.test(color)) return color;
  if (type === 'dashboard') return '#c338df';
  return 'var(--van-primary-color)';
}

function createDemoMenu(): MobileAppMenu {
  const names = [
    ['menu_order', '订单管理', 'form_order', 'workflow'],
    ['menu_purchase', '采购申请', 'form_purchase', 'standard'],
    ['menu_office', '办公用品申请', 'form_office', 'workflow'],
    ['menu_staff', '员工档案', 'form_staff', 'standard'],
    ['menu_product', '产品管理', 'form_product', 'standard'],
    ['menu_customer', '客户信息', 'form_customer', 'standard'],
  ] as const;
  const nodeMap: Record<string, MobileMenuNode> = {};
  names.forEach(([menuId, name, code, formType], index) => {
    nodeMap[menuId] = {
      menuId,
      parentMenuId: null,
      type: 'form',
      name,
      icon: null,
      color: index < 3 ? '#ff7a18' : null,
      sortOrder: index,
      target: { type: 'form', code, formType },
      capabilities: { view: true, favorite: true },
    };
  });
  ['员工信息分析', '订单分析'].forEach((name, index) => {
    const menuId = `menu_dashboard_${index}`;
    nodeMap[menuId] = {
      menuId,
      parentMenuId: null,
      type: 'dashboard',
      name,
      icon: null,
      color: '#c338df',
      sortOrder: names.length + index,
      target: { type: 'dashboard', code: `dashboard_${index}` },
      capabilities: { view: true, favorite: true },
    };
  });
  return {
    appCode: appCode.value,
    rootMenuIds: Object.keys(nodeMap),
    nodeMap,
    features: { workflow: true },
  };
}

async function load(): Promise<void> {
  loading.value = true;
  errorMessage.value = '';
  appName.value = typeof route.query.name === 'string' ? route.query.name : '应用';
  try {
    if (isDemo.value) {
      menu.value = createDemoMenu();
      return;
    }
    const [app, snapshot] = await Promise.all([
      getAppByCode(appCode.value),
      getAppMenuByCode(appCode.value),
    ]);
    appName.value = app.name;
    menu.value = snapshot;
  } catch (error) {
    menu.value = null;
    errorMessage.value = error instanceof Error ? error.message : '应用菜单加载失败';
  } finally {
    loading.value = false;
  }
}

function openMenu(item: MenuListItem): void {
  if (item.type === 'group') return;
  if (item.type !== 'form' || !item.targetCode) {
    showToast('该类型暂不支持在移动端打开');
    return;
  }
  void router.push({
    name: 'mobile-form',
    params: { appCode: appCode.value, formCode: item.targetCode },
    query: {
      menuCode: item.menuCode,
      name: item.name,
      formType: item.formType ?? 'standard',
    },
  });
}

watch(appCode, () => void load(), { immediate: true });
</script>

<template>
  <main class="app-page">
    <header class="app-brand-header">
      <button type="button" aria-label="返回工作台" @click="router.back()">
        <VanIcon name="arrow-left" />
      </button>
      <strong><i aria-hidden="true" />{{ tenantName }}</strong>
      <span />
    </header>

    <section class="app-title-row">
      <h1>{{ appName }}</h1>
      <button type="button" :aria-label="favorited ? '取消收藏' : '收藏应用'" @click="favorited = !favorited">
        <VanIcon :name="favorited ? 'star' : 'star-o'" />
      </button>
    </section>

    <section v-if="activeTab === 'directory'" class="app-directory">
      <label class="app-directory__search">
        <VanIcon name="search" />
        <input v-model="keyword" type="search" placeholder="输入名称来搜索">
      </label>

      <div v-if="loading" class="app-directory__state">
        <VanLoading color="var(--van-primary-color)" />
      </div>
      <div v-else-if="errorMessage" class="app-directory__state app-directory__state--error">
        <p>{{ errorMessage }}</p>
        <button type="button" @click="load">
          重新加载
        </button>
      </div>
      <VanEmpty v-else-if="menuItems.length === 0" image="search" description="暂无可用菜单" />
      <div v-else class="app-menu-list">
        <button
          v-for="item in menuItems"
          :key="item.menuCode"
          type="button"
          :class="{ 'app-menu-list__group': item.type === 'group' }"
          :style="{ paddingLeft: `${24 + item.depth * 22}px` }"
          @click="openMenu(item)"
        >
          <VanIcon :name="item.icon" :color="item.color" />
          <span>{{ item.name }}</span>
          <VanIcon v-if="item.type !== 'group'" class="app-menu-list__more" name="ellipsis" />
        </button>
      </div>
    </section>

    <MobileWorkflowCenter v-else :demo="isDemo" />

    <nav class="app-tabs" aria-label="应用导航">
      <button type="button" :class="{ 'is-active': activeTab === 'directory' }" @click="activeTab = 'directory'">
        <VanIcon name="orders-o" />
        <span>目录</span>
      </button>
      <button type="button" :class="{ 'is-active': activeTab === 'workflow' }" @click="activeTab = 'workflow'">
        <VanIcon name="cluster-o" />
        <span>流程中心</span>
      </button>
    </nav>
  </main>
</template>

<style scoped>
.app-page {
  display: flex;
  width: min(100%, 604px);
  height: 100dvh;
  min-height: 0;
  margin: 0 auto;
  color: #1d293b;
  background: #fff;
  flex-direction: column;
}

.app-brand-header {
  display: grid;
  min-height: calc(88px + env(safe-area-inset-top));
  padding: env(safe-area-inset-top) 16px 0;
  background: #f5f6f8;
  grid-template-columns: 44px 1fr 44px;
  align-items: end;
}

.app-brand-header > button {
  display: grid;
  width: 40px;
  height: 54px;
  padding: 0;
  font-size: 20px;
  color: #566171;
  background: none;
  border: 0;
  place-items: center;
}

.app-brand-header strong {
  display: flex;
  height: 54px;
  gap: 10px;
  font-size: 20px;
  font-weight: 500;
  align-items: center;
  justify-content: center;
}

.app-brand-header strong i {
  width: 18px;
  height: 18px;
  background: color-mix(in srgb, var(--van-primary-color) 72%, white);
  border-radius: 50%;
}

.app-title-row {
  display: flex;
  min-height: 82px;
  padding: 0 17px;
  border-bottom: 20px solid #f4f5f7;
  align-items: center;
}

.app-title-row h1 {
  margin: 0;
  overflow: hidden;
  font-size: 23px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-title-row button {
  padding: 10px;
  margin-left: 6px;
  font-size: 26px;
  color: var(--van-primary-color);
  background: none;
  border: 0;
}

.app-directory {
  min-height: 0;
  overflow-y: auto;
  flex: 1;
}

.app-directory__search {
  display: flex;
  height: 54px;
  margin: 14px 17px 19px;
  padding: 0 20px;
  gap: 13px;
  font-size: 21px;
  color: #8d96a3;
  background: #f5f6f8;
  border-radius: 28px;
  align-items: center;
}

.app-directory__search input {
  min-width: 0;
  font-size: 19px;
  color: #273247;
  background: none;
  border: 0;
  outline: 0;
  flex: 1;
}

.app-directory__search input::placeholder {
  color: #b1b7c0;
}

.app-directory__state {
  display: grid;
  min-height: 320px;
  place-items: center;
}

.app-directory__state--error {
  align-content: center;
  color: #7c8593;
}

.app-directory__state--error button {
  padding: 8px 18px;
  color: var(--van-primary-color);
  background: none;
  border: 1px solid var(--van-primary-color);
  border-radius: 18px;
}

.app-menu-list {
  padding-left: 0;
}

.app-menu-list > button {
  position: relative;
  box-sizing: border-box;
  display: flex;
  width: 100%;
  min-height: 85px;
  padding-right: 24px;
  gap: 27px;
  font-size: 22px;
  color: #253146;
  text-align: left;
  background: #fff;
  border: 0;
  align-items: center;
}

.app-menu-list > button::after {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 81px;
  height: 1px;
  content: '';
  background: #e7e9ed;
}

.app-menu-list > button > .van-icon:first-child {
  width: 30px;
  font-size: 30px;
  text-align: center;
}

.app-menu-list > button > span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.app-menu-list .app-menu-list__more {
  font-size: 28px;
  color: #89929f;
}

.app-menu-list__group {
  min-height: 54px !important;
  font-size: 16px !important;
  font-weight: 600;
  background: #f7f8fa !important;
}

.app-tabs {
  display: grid;
  flex: 0 0 auto;
  height: calc(94px + env(safe-area-inset-bottom));
  padding-bottom: env(safe-area-inset-bottom);
  background: #fff;
  border-top: 1px solid #e5e8ec;
  grid-template-columns: repeat(2, 1fr);
}

.app-tabs button {
  display: flex;
  padding: 10px 0 7px;
  gap: 6px;
  font-size: 15px;
  color: #596473;
  background: none;
  border: 0;
  flex-direction: column;
  align-items: center;
}

.app-tabs .van-icon {
  font-size: 32px;
}

.app-tabs button.is-active {
  color: var(--van-primary-color);
}

@media (max-width: 420px) {
  .app-brand-header {
    min-height: calc(68px + env(safe-area-inset-top));
  }

  .app-title-row {
    min-height: 70px;
    border-bottom-width: 14px;
  }

  .app-menu-list > button {
    min-height: 70px;
    gap: 18px;
    font-size: 18px;
  }

  .app-tabs {
    height: calc(70px + env(safe-area-inset-bottom));
  }

  .app-tabs .van-icon {
    font-size: 26px;
  }
}
</style>
