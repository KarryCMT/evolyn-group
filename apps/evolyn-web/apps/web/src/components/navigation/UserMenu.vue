<script setup lang="ts">
import { computed, shallowRef } from 'vue';
import { useRouter } from 'vue-router';
// 只显式导入 ElMessage：API 无模板标签可按需解析，样式由 main.ts 全局引入；
// el-dropdown / el-avatar 等组件必须走模板标签由 unplugin-vue-components 按需注入
// 组件与样式，显式 import 会绕过 resolver 导致组件样式丢失
import { ElMessage } from 'element-plus';
import { UserFilled } from '@element-plus/icons-vue';
import {
  RiArrowRightSFill,
  RiComputerFill,
  RiGlobalFill,
  RiLogoutBoxFill,
  RiSettings3Fill,
  RiStarFill,
  RiVipDiamondFill,
} from '@remixicon/vue';
import { useAuth } from '~/composables';
import FavoritesWorkspaceDialog from '~/components/dashboard/favorites/FavoritesWorkspaceDialog.vue';

defineOptions({ name: 'UserMenu' });

const router = useRouter();
const { userInfo, displayName, isTenantOwner, logout } = useAuth();

// 面板兜底名：聚合信息未拉到（或昵称为空）时仍保证信息区结构完整
const panelName = computed(() => displayName.value || '用户');

// 公司名：超长由样式截断为省略号（对齐设计稿「重庆万柯互联网科技有限责任...」形态）
const tenantName = computed(() => userInfo.value?.tenant.name ?? '');
// 账号已保存头像地址时优先渲染图片；空值才回退为 Element Plus 的用户图标。
const avatar = computed(() => userInfo.value?.account.avatar || '');
const favoritesVisible = shallowRef(false);

/**
 * 菜单指令分发：已落地页面真实跳转；
 * 版本购买和语言暂无对应页面，占位提示待后续里程碑落地。
 */
function onCommand(command: string | number | object) {
  switch (command) {
    case 'favorites':
      favoritesVisible.value = true;
      break;
    case 'settings':
      router.push({ name: 'account' });
      break;
    case 'admin':
      router.push({ name: 'tenant' });
      break;
    case 'logout':
      void handleLogout();
      break;
    default:
      ElMessage.info('功能建设中，敬请期待');
  }
}

/** 退出登录：清理会话后回登录页（接口失败也已在 store 内兜底清理） */
async function handleLogout() {
  await logout();
  router.push({ name: 'login' });
}
</script>

<template>
  <el-dropdown placement="bottom-end" popper-class="user-menu-popper" @command="onCommand">
    <!-- 触发器：顶栏小头像 -->
    <el-avatar class="user-menu__trigger" :size="24" :src="avatar" :icon="UserFilled" />
    <template #dropdown>
      <div class="user-menu">
        <!-- 用户信息区：头像 + 昵称/「我创建的」标签 + 公司名 -->
        <div class="user-menu__profile">
          <el-avatar class="user-menu__avatar" :size="40" :src="avatar" :icon="UserFilled" />
          <div class="user-menu__meta">
            <div class="user-menu__name-row">
              <span class="user-menu__name">{{ panelName }}</span>
              <span v-if="isTenantOwner" class="user-menu__owner-tag">我创建的</span>
            </div>
            <p class="user-menu__tenant">{{ tenantName }}</p>
          </div>
        </div>

        <div class="user-menu__separator" />

        <!-- 菜单列表：沿用 el-dropdown-item 保留 EP 的 command 派发与点击收起行为 -->
        <el-dropdown-menu class="user-menu__list">
          <el-dropdown-item command="favorites">
            <el-icon class="user-menu__item-icon"><RiStarFill /></el-icon>
            <span>我的收藏</span>
          </el-dropdown-item>
          <el-dropdown-item command="settings">
            <el-icon class="user-menu__item-icon"><RiSettings3Fill /></el-icon>
            <span>个人设置</span>
          </el-dropdown-item>
          <el-dropdown-item command="admin">
            <el-icon class="user-menu__item-icon"><RiComputerFill /></el-icon>
            <span>管理后台</span>
          </el-dropdown-item>
          <el-dropdown-item command="purchase">
            <el-icon class="user-menu__item-icon"><RiVipDiamondFill /></el-icon>
            <span>版本购买</span>
          </el-dropdown-item>
          <el-dropdown-item command="language">
            <el-icon class="user-menu__item-icon"><RiGlobalFill /></el-icon>
            <span>语言</span>
            <span class="user-menu__item-extra">简体中文</span>
            <el-icon class="user-menu__item-arrow"><RiArrowRightSFill /></el-icon>
          </el-dropdown-item>
          <!-- divided：借 EP 内置分隔线与常规菜单分区 -->
          <el-dropdown-item class="user-menu__item--danger" command="logout" divided>
            <el-icon class="user-menu__item-icon"><RiLogoutBoxFill /></el-icon>
            <span>退出</span>
          </el-dropdown-item>
        </el-dropdown-menu>
      </div>
    </template>
  </el-dropdown>
  <FavoritesWorkspaceDialog v-model="favoritesVisible" />
</template>

<style scoped lang="scss">
/* 触发器头像：el-avatar 默认无交互态，补指针提示 */
.user-menu__trigger {
  cursor: pointer;
  transition: box-shadow 0.18s ease;

  &:hover {
    box-shadow: 0 0 0 4px rgb(54 65 82 / 8%);
  }
}
</style>

<!-- popper 默认传送至 body，scoped 样式作用不到，面板样式须全局书写；
     全部选择器以 .user-menu-popper / .user-menu 前缀限定，避免泄漏到其他弹层 -->
<style lang="scss">
.user-menu-popper.el-popper {
  width: 250px;
  border-radius: 8px;
  border-color: var(--el-border-color-lighter);
  box-shadow: var(--el-box-shadow-light);
}

.user-menu__profile {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 16px 12px;
}

.user-menu__avatar {
  flex-shrink: 0;
  background: var(--el-color-primary-light-7);
  color: var(--el-color-primary);
}

/* meta 容器收窄最小宽度，配合内部省略号截断 */
.user-menu__meta {
  min-width: 0;
}

.user-menu__name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-menu__name {
  overflow: hidden;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-menu__owner-tag {
  flex-shrink: 0;
  padding: 0 6px;
  font-size: 12px;
  line-height: 18px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
  border-radius: 4px;
}

.user-menu__tenant {
  margin: 2px 0 0;
  overflow: hidden;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-menu__separator {
  height: 1px;
  margin: 0 12px;
  background: var(--el-border-color-lighter);
}

.user-menu__list.el-dropdown-menu {
  padding: 6px 8px 8px;

  /* 悬停态对齐设计稿：浅灰底 + 深色字（覆盖 EP 默认的主题蓝悬停） */
  --el-dropdown-menuItem-hover-fill: var(--el-fill-color-light);
  --el-dropdown-menuItem-hover-color: var(--el-text-color-primary);
}

.user-menu__list .el-dropdown-menu__item {
  height: 40px;
  gap: 10px;
  padding: 0 10px;
  border-radius: 6px;
}

/* 退出：红色文字/图标 + 红色悬停底，与常规菜单区分；
   悬停变量就地覆盖列表级定义，保证该项悬停仍为红色系 */
.user-menu__item--danger.el-dropdown-menu__item {
  color: var(--el-color-danger);
  --el-dropdown-menuItem-hover-fill: var(--el-color-danger-light-9);
  --el-dropdown-menuItem-hover-color: var(--el-color-danger);
}

.user-menu__item-icon {
  font-size: 16px;
}

.user-menu__item-extra {
  margin-left: auto;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.user-menu__item-arrow {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}
</style>
