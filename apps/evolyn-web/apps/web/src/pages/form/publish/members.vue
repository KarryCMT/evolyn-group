<script setup lang="ts">
import { RiAddFill, RiForbid2Fill, RiLinkM } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { shallowRef } from 'vue';

defineOptions({ name: 'FormPublishMembersPage' });

const permissionEnabled = shallowRef(true);

function notifyUnavailable(action: string) {
  ElMessage.info(`${action}将在成员权限接口接入后提供`);
}
</script>

<template>
  <section class="form-publish-members-page" aria-label="对成员发布">
    <header class="form-publish-members-page__header">
      <div>
        <div class="form-publish-members-page__heading-row">
          <h1 class="form-publish-members-page__title">对成员发布</h1>
          <p class="form-publish-members-page__description">
            设置成员权限，成员登录后将根据权限访问表单。
          </p>
        </div>
      </div>
      <button
        class="form-publish-members-page__access-link"
        type="button"
        @click="notifyUnavailable('表单访问链接')"
      >
        <RiLinkM />
        表单访问链接
      </button>
    </header>

    <div class="form-publish-members-page__body">
      <div class="form-publish-members-page__actions">
        <button
          class="form-publish-members-page__add-button"
          type="button"
          @click="notifyUnavailable('添加成员')"
        >
          <RiAddFill />
          添加成员
        </button>
        <button
          class="form-publish-members-page__disable-button"
          type="button"
          @click="notifyUnavailable('停用全部权限')"
        >
          <RiForbid2Fill />
          停用全部
        </button>
      </div>

      <section class="form-publish-members-page__permission-card">
        <header class="form-publish-members-page__permission-header">
          <div>
            <h2 class="form-publish-members-page__permission-title">管理全部数据</h2>
            <p class="form-publish-members-page__permission-description">
              此分组内的成员可以管理全部数据、填报数据，但不可以导入数据。
            </p>
          </div>
          <div class="form-publish-members-page__permission-actions">
            <button type="button" @click="notifyUnavailable('编辑权限')">编辑</button>
            <button type="button" @click="notifyUnavailable('复制权限')">复制</button>
            <button type="button" @click="notifyUnavailable('其他权限设置')">其他设置</button>
            <button
              class="form-publish-members-page__delete-button"
              type="button"
              @click="notifyUnavailable('删除权限')"
            >
              删除
            </button>
            <el-switch v-model="permissionEnabled" />
          </div>
        </header>

        <button
          class="form-publish-members-page__member-picker"
          type="button"
          @click="notifyUnavailable('选择成员或部门')"
        >
          <RiAddFill />
          选择成员或部门
        </button>
      </section>
    </div>
  </section>
</template>

<style scoped lang="scss">
.form-publish-members-page {
  min-height: 100%;

  &__header,
  &__heading-row,
  &__actions,
  &__permission-header,
  &__permission-actions,
  &__add-button,
  &__disable-button,
  &__access-link,
  &__member-picker {
    display: flex;
    align-items: center;
  }

  &__header {
    min-height: 68px;
    padding: 0 28px;
    justify-content: space-between;
    gap: 20px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__heading-row {
    gap: 16px;
  }

  &__title,
  &__description,
  &__permission-title,
  &__permission-description {
    margin: 0;
  }

  &__title,
  &__permission-title {
    color: var(--el-text-color-primary);
  }

  &__title {
    font-size: 19px;
    line-height: 28px;
  }

  &__description,
  &__permission-description {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  &__access-link,
  &__permission-actions button {
    padding: 0;
    color: var(--el-color-primary);
    background: transparent;
    border: 0;
    cursor: pointer;

    &:hover {
      color: var(--el-color-primary-light-3);
      background: var(--el-color-primary-light-9);
    }
  }

  &__access-link {
    min-height: 32px;
    padding: 0 8px;
    gap: 6px;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 18px;
      height: 18px;
    }
  }

  &__body {
    max-width: 1120px;
    padding: 28px;
  }

  &__actions {
    justify-content: space-between;
  }

  &__add-button,
  &__disable-button,
  &__member-picker {
    border: 0;
    cursor: pointer;
    font-weight: 600;

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__add-button {
    height: 40px;
    padding: 0 16px;
    gap: 7px;
    color: var(--el-color-white);
    background: var(--el-color-primary);
    border-radius: var(--el-border-radius-base);

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      background: var(--el-color-primary-light-3);
    }
  }

  &__disable-button {
    min-height: 32px;
    padding: 0 8px;
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--el-color-danger);
    background: transparent;
    border-radius: var(--el-border-radius-base);

    &:hover {
      background: var(--el-color-danger-light-9);
    }
  }

  &__permission-card {
    margin-top: 36px;
  }

  &__permission-header {
    justify-content: space-between;
    gap: 20px;
  }

  &__permission-title {
    font-size: 16px;
    line-height: 24px;
  }

  &__permission-description {
    margin-top: 6px;
    line-height: 22px;
  }

  &__permission-actions {
    gap: 10px;
    white-space: nowrap;

    button {
      min-height: 30px;
      padding: 0 6px;
      border-radius: var(--el-border-radius-base);
    }
  }

  &__delete-button {
    color: var(--el-color-danger) !important;

    &:hover {
      color: var(--el-color-danger-light-3) !important;
      background: var(--el-color-danger-light-9) !important;
    }
  }

  &__member-picker {
    width: 100%;
    min-height: 120px;
    margin-top: 10px;
    justify-content: center;
    gap: 8px;
    color: var(--el-text-color-regular);
    background: transparent;
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-base);
    font-size: 16px;

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
      border-color: var(--el-color-primary);
    }
  }
}

@media (max-width: 900px) {
  .form-publish-members-page {
    &__header,
    &__permission-header {
      padding: 18px;
      align-items: flex-start;
      flex-direction: column;
    }

    &__body {
      padding: 24px 18px;
    }

    &__permission-actions {
      flex-wrap: wrap;
      white-space: normal;
    }
  }
}
</style>
