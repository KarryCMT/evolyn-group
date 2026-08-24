<script setup lang="ts">
import { RiArrowDownSFill, RiCloseFill, RiOrganizationChart, RiSearchFill } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import type { PermissionMember } from './permissionQuery.types';

defineOptions({ name: 'PermissionQueryMembersDialog' });

const props = defineProps<{
  members: PermissionMember[];
  selectedMembers: PermissionMember[];
}>();

const visible = defineModel<boolean>({ default: false });
const emit = defineEmits<{ confirm: [members: PermissionMember[]] }>();
const keyword = shallowRef('');
const selectedIds = shallowRef<string[]>([]);
const selectedMembers = computed(() =>
  props.members.filter((member) => selectedIds.value.includes(member.id)),
);
const visibleMembers = computed(() => {
  const query = keyword.value.trim();
  return query
    ? props.members.filter((member) => `${member.name}${member.department}`.includes(query))
    : props.members;
});

function remove(id: string) {
  selectedIds.value = selectedIds.value.filter((item) => item !== id);
}

function toggle(id: string) {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter((item) => item !== id)
    : [...selectedIds.value, id];
}

function submit() {
  emit('confirm', selectedMembers.value);
  visible.value = false;
}

watch(visible, (isVisible) => {
  if (isVisible) {
    keyword.value = '';
    selectedIds.value = props.selectedMembers.map((member) => member.id);
  }
});
</script>

<template>
  <el-dialog
    v-model="visible"
    class="permission-query-members-dialog"
    width="1032px"
    :show-close="false"
    append-to-body
  >
    <header class="permission-query-members-dialog__header">
      <h2>成员列表</h2>
      <button type="button" aria-label="关闭" @click="visible = false"><RiCloseFill /></button>
    </header>
    <section class="permission-query-members-dialog__selected" aria-label="已选成员">
      <span
        v-for="member in selectedMembers"
        :key="member.id"
        class="permission-query-members-dialog__tag"
      >
        <i>{{ member.name.slice(0, 1) }}</i
        ><span>{{ member.name }}</span
        ><RiCloseFill @click="remove(member.id)" />
      </span>
    </section>
    <label class="permission-query-members-dialog__search">
      <RiSearchFill /><input v-model="keyword" placeholder="搜索（多个关键词用空格隔开）" />
    </label>
    <div class="permission-query-members-dialog__body">
      <section class="permission-query-members-dialog__tree">
        <p class="permission-query-members-dialog__all">全部成员</p>
        <p class="permission-query-members-dialog__node">
          <RiArrowDownSFill /><RiOrganizationChart />重庆万柯互联网科技有限责任公司
        </p>
        <p class="permission-query-members-dialog__child"><RiOrganizationChart />研发部</p>
        <p class="permission-query-members-dialog__child"><RiOrganizationChart />产品部</p>
      </section>
      <section class="permission-query-members-dialog__results">
        <p>已选 {{ selectedMembers.length }}/{{ props.members.length }}</p>
        <label
          v-for="member in visibleMembers"
          :key="member.id"
          class="permission-query-members-dialog__member"
        >
          <span class="permission-query-members-dialog__avatar">{{ member.name.slice(0, 1) }}</span
          ><span>{{ member.name }}</span>
          <el-checkbox :model-value="selectedIds.includes(member.id)" @change="toggle(member.id)" />
        </label>
      </section>
    </div>
    <footer class="permission-query-members-dialog__footer">
      <el-button @click="visible = false">取消</el-button
      ><el-button type="primary" @click="submit">确定</el-button>
    </footer>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.permission-query-members-dialog) {
  border-radius: 14px;
}
:global(.permission-query-members-dialog .el-dialog__header) {
  display: none;
}
:global(.permission-query-members-dialog .el-dialog__body) {
  padding: 0;
}
.permission-query-members-dialog {
  &__header,
  &__footer {
    display: flex;
    height: 68px;
    padding: 0 28px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    align-items: center;
    justify-content: space-between;
  }
  &__header h2 {
    margin: 0;
    font-size: 22px;
  }
  &__header button {
    display: inline-flex;
    padding: 4px;
    border: 0;
    color: var(--el-text-color-regular);
    background: transparent;
    cursor: pointer;
  }
  &__header button:hover {
    border-radius: 6px;
    background: var(--el-fill-color-light);
  }
  &__header svg {
    width: 24px;
    height: 24px;
  }
  &__selected {
    display: flex;
    min-height: 132px;
    margin: 26px 28px 16px;
    padding: 12px;
    border: 1px dashed var(--el-border-color);
    border-radius: 8px;
    align-items: flex-start;
    gap: 8px;
  }
  &__tag {
    display: inline-flex;
    height: 42px;
    gap: 7px;
    padding: 0 12px;
    border-radius: 7px;
    align-items: center;
    background: var(--el-fill-color);
  }
  &__tag i,
  &__avatar {
    display: inline-flex;
    width: 26px;
    height: 26px;
    border-radius: 50%;
    align-items: center;
    justify-content: center;
    color: #fff;
    background: var(--el-color-danger);
    font-size: 14px;
    font-style: normal;
  }
  &__tag svg {
    margin-left: 42px;
    cursor: pointer;
  }
  &__tag svg:hover {
    color: var(--el-color-danger);
  }
  &__search {
    display: flex;
    height: 44px;
    margin: 0 28px;
    padding: 0 12px;
    border-radius: 7px;
    align-items: center;
    gap: 8px;
    background: var(--el-fill-color-light);
  }
  &__search svg {
    width: 20px;
    height: 20px;
    color: var(--el-text-color-secondary);
  }
  &__search input {
    width: 100%;
    border: 0;
    outline: 0;
    color: var(--el-text-color-primary);
    background: transparent;
    font: inherit;
  }
  &__body {
    display: grid;
    min-height: 460px;
    margin: 6px 28px 0;
    grid-template-columns: 1fr 1fr;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  &__tree,
  &__results {
    padding: 14px 12px;
  }
  &__tree {
    border-right: 1px solid var(--el-border-color-lighter);
  }
  &__all {
    margin: 0 0 9px;
    padding: 12px 16px;
    border-radius: 7px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__node,
  &__child {
    display: flex;
    margin: 12px;
    align-items: center;
    gap: 8px;
  }
  &__node svg,
  &__child svg {
    color: var(--el-color-primary);
  }
  &__child {
    margin-left: 58px;
  }
  &__results p {
    margin: 0 0 12px;
    color: var(--el-color-primary);
  }
  &__member {
    display: flex;
    height: 43px;
    align-items: center;
    gap: 8px;
    cursor: pointer;
  }
  &__member .el-checkbox {
    margin-left: auto;
  }
  &__footer {
    height: 76px;
    border-top: 1px solid var(--el-border-color-lighter);
    border-bottom: 0;
    justify-content: flex-end;
    gap: 12px;
  }
  &__footer .el-button {
    min-width: 74px;
    height: 42px;
  }
}
</style>
