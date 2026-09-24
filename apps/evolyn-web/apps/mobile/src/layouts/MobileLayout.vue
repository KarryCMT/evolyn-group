<script setup lang="ts">
import {
  NavBar as VanNavBar,
  Tabbar as VanTabbar,
  TabbarItem as VanTabbarItem,
} from 'vant';
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';

const route = useRoute();
const router = useRouter();
const activeTab = computed(() => route.meta.tab ?? '');
const showTabbar = computed(() => Boolean(route.meta.tab));

function goBack(): void {
  if (window.history.length > 1) router.back();
  else void router.replace('/');
}
</script>

<template>
  <div class="mobile-layout">
    <VanNavBar
      class="mobile-layout__header"
      :title="route.meta.title"
      :left-arrow="route.meta.showBack"
      fixed
      placeholder
      safe-area-inset-top
      @click-left="goBack"
    />
    <main class="mobile-layout__content" :class="{ 'mobile-layout__content--with-tabs': showTabbar }">
      <RouterView />
    </main>
    <VanTabbar v-if="showTabbar" :model-value="activeTab" route fixed safe-area-inset-bottom>
      <VanTabbarItem name="home" to="/" replace icon="apps-o">
        工作台
      </VanTabbarItem>
      <VanTabbarItem name="tasks" to="/tasks" replace icon="todo-list-o">
        待办
      </VanTabbarItem>
      <VanTabbarItem name="me" to="/me" replace icon="user-o">
        我的
      </VanTabbarItem>
    </VanTabbar>
  </div>
</template>

<style scoped>
.mobile-layout {
  min-height: 100dvh;
  background: var(--van-background);
}

.mobile-layout__header {
  --van-nav-bar-background: color-mix(in srgb, var(--van-background-2) 94%, transparent);
}

.mobile-layout__content {
  min-height: calc(100dvh - var(--van-nav-bar-height));
}

.mobile-layout__content--with-tabs {
  padding-bottom: calc(var(--van-tabbar-height) + env(safe-area-inset-bottom));
}
</style>
