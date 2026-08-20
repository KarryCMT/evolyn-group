<script setup lang="ts">
import { ElAvatar, ElTooltip } from 'element-plus';
import { ArrowLeft, Bell, Grid, QuestionFilled, SetUp, UserFilled } from '@element-plus/icons-vue';
import { RiHomeGearFill,RiGridFill } from "@remixicon/vue";
import { useRouter } from 'vue-router';

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
      <el-icon v-if="backTo" size="16" aria-label="返回工作台" @click="goBack" >
        <RiGridFill />
      </el-icon>
      <template v-else>
        <el-button class="top-navigation__switcher" circle :icon="Grid" aria-label="切换产品" />
        <span class="top-navigation__logo" aria-hidden="true">

        </span>
      </template>
      <strong>{{ title }}</strong>
    </div>

    <nav class="top-navigation__actions" aria-label="工作台导航">
      <template v-if="!backTo">
        <el-tooltip content="自定义工作台" placement="bottom">
          <el-icon
            class="top-navigation__workbench-entry"
            @click="openWorkbenchEditor"
          >
            <RiHomeGearFill />
          </el-icon>
        </el-tooltip>
        <el-button text>模板中心</el-button>
        <el-button text>通讯录</el-button>
      </template>
      <el-button text circle :icon="Bell" aria-label="通知" />
      <el-button v-if="backTo" text circle :icon="QuestionFilled" aria-label="帮助" />
      <el-dropdown>
        <el-avatar :size="28" :icon="UserFilled" />
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item>个人中心</el-dropdown-item>
            <el-dropdown-item>退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </nav>
  </header>
</template>

<style scoped lang="scss">
.top-navigation {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 52px;
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
