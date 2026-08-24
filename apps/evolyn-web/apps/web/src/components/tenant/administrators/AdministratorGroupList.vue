<script setup lang="ts">
import { RiAddFill, RiArrowDownSFill, RiGroupFill, RiUserSettingsFill } from '@remixicon/vue';
import type { AdministratorGroup, AdministratorScope } from './administrator.types';

defineOptions({ name: 'AdministratorGroupList' });

const props = defineProps<{
  scope: AdministratorScope;
  groups: AdministratorGroup[];
  selectedId: string;
}>();

const emit = defineEmits<{
  select: [id: string];
  add: [];
}>();

const groupTitle = props.scope === 'system' ? '系统管理组' : '普通管理组';
</script>

<template>
  <aside class="administrator-group-list" aria-label="管理组列表">
    <section v-if="scope === 'system'" class="administrator-group-list__built-in">
      <p class="administrator-group-list__section-title">{{ groupTitle }}</p>
      <button
        v-for="group in groups.filter((item) => item.builtIn)"
        :key="group.id"
        class="administrator-group-list__item"
        :class="{ 'administrator-group-list__item--active': group.id === selectedId }"
        type="button"
        @click="emit('select', group.id)"
      >
        <RiUserSettingsFill />
        <span>{{ group.name }}</span>
      </button>
    </section>

    <section class="administrator-group-list__custom">
      <div class="administrator-group-list__custom-heading">
        <p class="administrator-group-list__section-title">
          {{ scope === 'system' ? '通讯录管理组' : groupTitle }}
        </p>
        <button
          class="administrator-group-list__add"
          type="button"
          aria-label="添加管理组"
          @click="emit('add')"
        >
          <RiAddFill />
        </button>
      </div>
      <button class="administrator-group-list__select" type="button" aria-label="选择管理组">
        <RiGroupFill />
        <RiArrowDownSFill />
        <span>选择管理组</span>
        <RiArrowDownSFill />
      </button>
      <button
        v-for="group in groups.filter((item) => !item.builtIn)"
        :key="group.id"
        class="administrator-group-list__item"
        :class="{ 'administrator-group-list__item--active': group.id === selectedId }"
        type="button"
        @click="emit('select', group.id)"
      >
        <RiGroupFill />
        <span>{{ group.name }}</span>
      </button>
    </section>
  </aside>
</template>

<style scoped lang="scss">
.administrator-group-list {
  box-sizing: border-box;
  display: flex;
  min-width: 356px;
  height: 100%;
  flex-direction: column;
  padding: 30px 28px;
  border-right: 1px solid #e5e8ee;
  background: #fff;
  color: #5c6472;

  &__built-in,
  &__custom {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  &__custom {
    margin-top: 30px;
  }

  &__custom-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &__section-title {
    margin: 0;
    color: #818a98;
    font-size: 18px;
    line-height: 32px;
  }

  &__add {
    display: inline-flex;
    width: 36px;
    height: 36px;
    padding: 0;
    border: 0;
    border-radius: 9px;
    align-items: center;
    justify-content: center;
    color: #fff;
    background: var(--el-color-primary);
    cursor: pointer;

    svg {
      width: 24px;
      height: 24px;
    }

    &:hover {
      background: var(--el-color-primary-light-3);
    }
  }

  &__select {
    display: grid;
    height: 44px;
    grid-template-columns: 30px 20px 1fr 20px;
    padding: 0 12px;
    border: 1px solid #dce2eb;
    border-radius: 7px;
    align-items: center;
    color: #8b94a2;
    background: #fff;
    font-size: 17px;
    text-align: left;
    cursor: pointer;

    svg {
      width: 21px;
      height: 21px;
    }

    &:hover {
      border-color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__item {
    display: flex;
    width: 100%;
    height: 56px;
    gap: 10px;
    padding: 0 18px;
    border: 0;
    border-radius: 8px;
    align-items: center;
    color: #626b78;
    background: transparent;
    font-size: 18px;
    text-align: left;
    cursor: pointer;

    svg {
      width: 24px;
      height: 24px;
      color: #9ca5b3;
    }

    &:hover {
      background: var(--el-color-primary-light-9);
      color: var(--el-color-primary);
    }

    &--active {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
    &--active svg {
      color: var(--el-color-primary);
    }
  }
}
</style>
