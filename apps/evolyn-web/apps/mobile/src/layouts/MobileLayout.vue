<script setup lang="ts">
import { NavBar as VanNavBar } from 'vant';
import { useRoute, useRouter } from 'vue-router';

const route = useRoute();
const router = useRouter();

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
    <main class="mobile-layout__content">
      <RouterView />
    </main>
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

</style>
