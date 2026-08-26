<script setup lang="ts">
import type { DeepReadonly } from 'vue';
import type { UserInfoResult } from '~/types';
import { RiCheckboxCircleFill, RiUserFill } from '@remixicon/vue';
import { computed, nextTick, ref, useTemplateRef, watch } from 'vue';
import type { InputInstance } from 'element-plus';

defineOptions({ name: 'AccountBasicInfoPanel' });

const emit = defineEmits<{
  bindEmail: [];
  editAvatar: [file: File];
  updateContactName: [nickname: string, onSuccess: () => void];
  changePassword: [];
  viewLoginLog: [];
}>();

const props = defineProps<{
  userInfo: DeepReadonly<UserInfoResult> | null;
  savingContactName?: boolean;
}>();

const contactNameInputRef = ref<InputInstance>();
const avatarInputRef = useTemplateRef<HTMLInputElement>('avatarInput');
const contactNameEditing = ref(false);
const contactName = ref('');

const displayedContactName = computed(
  () => props.userInfo?.member.nickname || props.userInfo?.account.nickname || '未设置',
);
const contactNameError = computed(() => {
  const name = contactName.value.trim();
  if (!name) return '姓名不能为空';
  if (name.length > 80) return '姓名不能超过80个字符';
  return '';
});

// 外部资料刷新后，同步展示值；编辑中的输入保留用户尚未提交的内容。
watch(
  displayedContactName,
  (name) => {
    if (!contactNameEditing.value) contactName.value = name === '未设置' ? '' : name;
  },
  { immediate: true },
);

async function startContactNameEditing() {
  contactName.value = displayedContactName.value === '未设置' ? '' : displayedContactName.value;
  contactNameEditing.value = true;
  await nextTick();
  contactNameInputRef.value?.focus();
  contactNameInputRef.value?.select();
}

function cancelContactNameEditing() {
  contactNameEditing.value = false;
  contactName.value = displayedContactName.value === '未设置' ? '' : displayedContactName.value;
}

function submitContactName() {
  if (contactNameError.value || props.savingContactName) return;
  emit('updateContactName', contactName.value.trim(), () => {
    contactNameEditing.value = false;
  });
}

// 头像编辑遵循「先选择文件，再打开裁剪弹窗」的交互，取消系统选图时不打断当前页面。
function chooseAvatar() {
  avatarInputRef.value?.click();
}

function handleAvatarChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (file) emit('editAvatar', file);
}
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
          <el-button link type="primary" @click="chooseAvatar">修改</el-button>
          <input
            ref="avatarInput"
            class="account-basic-info__avatar-input"
            type="file"
            accept="image/jpeg,image/png,.jpg,.jpeg,.png"
            @change="handleAvatarChange"
          />
        </dd>
      </div>
      <div class="account-basic-info__row">
        <dt>通讯录姓名</dt>
        <dd class="account-basic-info__contact-name">
          <template v-if="contactNameEditing">
            <div class="account-basic-info__contact-name-editor">
              <div class="account-basic-info__contact-name-tips">
                <span>
                  <el-icon><RiCheckboxCircleFill /></el-icon>
                  姓名不能为空
                </span>
                <span>
                  <el-icon><RiCheckboxCircleFill /></el-icon>
                  不能超过80个字符
                </span>
              </div>
              <div class="account-basic-info__contact-name-actions">
                <el-input
                  ref="contactNameInputRef"
                  v-model="contactName"
                  maxlength="80"
                  autocomplete="nickname"
                  @keydown.enter.prevent="submitContactName"
                  @keydown.esc.prevent="cancelContactNameEditing"
                />
                <el-button type="primary" :loading="savingContactName" @click="submitContactName">
                  确定
                </el-button>
                <el-button :disabled="savingContactName" @click="cancelContactNameEditing"
                  >取消</el-button
                >
              </div>
            </div>
          </template>
          <template v-else>
            <span>{{ displayedContactName }}</span>
            <el-button link type="primary" @click="startContactNameEditing">修改</el-button>
          </template>
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
          <el-button link type="primary" @click="emit('bindEmail')">
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

  &__contact-name {
    min-height: 54px;
  }

  &__contact-name-editor {
    width: min(100%, 520px);
    padding: 10px 0;
  }

  &__contact-name-tips {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 8px;
    color: var(--el-color-success);
    font-size: var(--el-font-size-small);
  }

  &__contact-name-tips span {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  &__contact-name-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__contact-name-actions :deep(.el-input) {
    min-width: 0;
    flex: 1;
  }

  &__identifier {
    font-family: var(--el-font-family);
    font-size: var(--el-font-size-small);
  }

  &__avatar-input {
    display: none;
  }
}
</style>
