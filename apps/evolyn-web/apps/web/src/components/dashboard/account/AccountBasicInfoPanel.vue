<script setup lang="ts">
import type { DeepReadonly } from 'vue';
import type { UserInfoResult } from '~/types';
import { RiUserFill } from '@remixicon/vue';

defineOptions({ name: 'AccountBasicInfoPanel' });

defineProps<{
  userInfo: DeepReadonly<UserInfoResult> | null;
}>();

const emit = defineEmits<{
  editProfile: [];
  changePassword: [];
  viewLoginLog: [];
}>();
</script>

<template>
  <section class="account-basic-info" aria-label="基本信息">
    <div class="account-basic-info__tenant">
      <strong>当前所在企业：{{ userInfo?.tenant.name || '暂未加载企业信息' }}</strong>
      <el-tag
        v-if="userInfo?.tenant.ownerAccountId === userInfo?.account.id"
        size="small"
        effect="plain"
      >
        我创建的
      </el-tag>
    </div>

    <dl class="account-basic-info__list">
      <div class="account-basic-info__row">
        <dt>通讯录头像</dt>
        <dd>
          <el-avatar :size="36" :src="userInfo?.account.avatar">
            <el-icon><RiUserFill /></el-icon>
          </el-avatar>
          <el-button link type="primary" @click="emit('editProfile')">修改</el-button>
        </dd>
      </div>
      <div class="account-basic-info__row">
        <dt>通讯录姓名</dt>
        <dd>
          <span>{{ userInfo?.member.nickname || userInfo?.account.nickname || '未设置' }}</span>
          <el-button link type="primary" @click="emit('editProfile')">修改</el-button>
        </dd>
      </div>
      <div class="account-basic-info__row">
        <dt>用户 ID</dt>
        <dd class="account-basic-info__identifier">{{ userInfo?.account.id ?? '--' }}</dd>
      </div>
      <div class="account-basic-info__row">
        <dt>登录日志</dt>
        <dd><el-button link type="primary" @click="emit('viewLoginLog')">查看</el-button></dd>
      </div>
    </dl>

    <el-divider />

    <dl class="account-basic-info__list account-basic-info__list--security">
      <div class="account-basic-info__row">
        <dt>密码</dt>
        <dd>
          <span>{{ userInfo?.account.passwordInitialized ? '已设置' : '尚未设置' }}</span>
          <el-button link type="primary" @click="emit('changePassword')">
            {{ userInfo?.account.passwordInitialized ? '修改' : '设置' }}
          </el-button>
        </dd>
      </div>
      <div class="account-basic-info__row">
        <dt>手机</dt>
        <dd>{{ userInfo?.account.phone || '未绑定' }}</dd>
      </div>
      <div class="account-basic-info__row">
        <dt>邮箱</dt>
        <dd>
          <span>{{ userInfo?.account.email || '未绑定' }}</span>
          <el-button link type="primary" @click="emit('editProfile')">
            {{ userInfo?.account.email ? '修改' : '绑定' }}
          </el-button>
        </dd>
      </div>
    </dl>
  </section>
</template>

<style scoped lang="scss">
.account-basic-info {
  &__tenant {
    display: flex;
    align-items: center;
    min-height: 40px;
    gap: 10px;
    margin-bottom: 22px;
    color: var(--el-text-color-primary);
  }

  &__list {
    margin: 0;
  }

  &__list--security {
    padding-top: 4px;
  }

  &__row {
    display: grid;
    grid-template-columns: 162px minmax(0, 1fr);
    align-items: center;
    min-height: 54px;
  }

  &__row dt {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__row dd {
    display: flex;
    align-items: center;
    min-width: 0;
    margin: 0;
    gap: 12px;
    color: var(--el-text-color-regular);
  }

  &__identifier {
    font-family: var(--el-font-family);
    font-size: var(--el-font-size-small);
  }
}
</style>
