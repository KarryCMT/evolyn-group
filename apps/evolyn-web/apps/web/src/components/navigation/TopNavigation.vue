<script setup lang="ts">
// el-tooltip 等组件走模板标签按需解析注入样式，显式 import EP 组件会绕过
// unplugin-vue-components 的样式注入导致组件无样式（详见 UserMenu.vue 注释）
import { Bell, Grid, QuestionFilled } from '@element-plus/icons-vue';
import { RiArrowLeftLine, RiHomeGearFill } from '@remixicon/vue';
import { useRouter } from 'vue-router';
import UserMenu from '~/components/navigation/UserMenu.vue';

defineOptions({ name: 'TopNavigation' });

const props = withDefaults(
  defineProps<{
    title?: string;
    backTo?: string;
  }>(),
  {
    title: '工作台',
    backTo: undefined,
  },
);

const router = useRouter();
function goBack() {
  if (props.backTo) router.push(props.backTo);
}

/** 成员端只提供入口，卡片编排始终在独立设计页完成。 */
function openWorkbenchEditor() {
  router.push({ name: 'custom_workbench' });
}
</script>

<template>
  <header class="top-navigation">
    <div class="top-navigation__brand">
      <el-icon v-if="backTo" size="16" aria-label="返回工作台" @click="goBack">
        <RiArrowLeftLine />
      </el-icon>
      <template v-else>
        <el-button class="top-navigation__switcher" circle :icon="Grid" aria-label="切换产品" />
        <span class="top-navigation__logo" aria-hidden="true"> </span>
      </template>
      <strong>{{ title }}</strong>
    </div>

    <nav class="top-navigation__actions" aria-label="工作台导航">
      <template v-if="!backTo">
        <el-tooltip content="自定义工作台" placement="bottom">
          <el-icon class="top-navigation__workbench-entry" @click="openWorkbenchEditor">
            <RiHomeGearFill />
          </el-icon>
        </el-tooltip>
        <el-button text>模板中心</el-button>
        <el-button text>通讯录</el-button>
      </template>
      <el-button text circle :icon="Bell" aria-label="通知" />
      <el-button v-if="backTo" text circle :icon="QuestionFilled" aria-label="帮助" />
      <!-- 用户头像下拉：信息区 + 菜单面板（简道云形态） -->
      <UserMenu />
    </nav>
  </header>
</template>

<style scoped lang="scss">
.top-navigation {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 52px;
  padding: 0 16px;
  color: var(--el-text-color-primary);
  background: #f3f3f8;

  &__brand,
  &__actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__switcher {
    color: var(--el-text-color-primary);
  }

  &__workbench-entry {
    margin-right: 4px;
    border-right: 1px solid var(--el-border-color-lighter);
    border-radius: 0;
    padding-right: 12px;
    &:hover {
      cursor: pointer;
      color: var(--el-color-primary);
    }
  }

  &__logo {
    color: var(--el-color-primary);
    font-size: var(--el-font-size-large);
  }
}
</style>
