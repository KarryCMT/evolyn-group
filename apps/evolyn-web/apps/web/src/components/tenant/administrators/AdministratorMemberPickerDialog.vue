<script setup lang="ts">
import { RiArrowDownSFill, RiCloseFill, RiSearch2Line, RiTeamFill } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import type { AdministratorMember } from './administrator.types';

defineOptions({ name: 'AdministratorMemberPickerDialog' });

const props = defineProps<{
  members: AdministratorMember[];
  selectedMembers: AdministratorMember[];
}>();

const visible = defineModel<boolean>({ default: false });
const emit = defineEmits<{ confirm: [members: AdministratorMember[]] }>();
const keyword = shallowRef('');
const selectedIds = shallowRef<string[]>([]);
const visibleMembers = computed(() => {
  const query = keyword.value.trim();
  return query
    ? props.members.filter((member) => `${member.name}${member.department}`.includes(query))
    : props.members;
});
const selectedItems = computed(() =>
  props.members.filter((member) => selectedIds.value.includes(member.id)),
);

function toggleMember(id: string) {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter((item) => item !== id)
    : [...selectedIds.value, id];
}

function removeMember(id: string) {
  selectedIds.value = selectedIds.value.filter((item) => item !== id);
}

function submit() {
  emit('confirm', selectedItems.value);
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
    class="administrator-member-picker"
    width="1034px"
    :show-close="false"
    append-to-body
  >
    <header class="administrator-member-picker__header">
      <h2>成员列表</h2>
      <button type="button" aria-label="关闭" @click="visible = false"><RiCloseFill /></button>
    </header>
    <section class="administrator-member-picker__selected" aria-label="已选成员">
      <span
        v-for="member in selectedItems"
        :key="member.id"
        class="administrator-member-picker__tag"
      >
        <i>{{ member.name.slice(0, 1) }}</i
        >{{ member.name }}<RiCloseFill @click="removeMember(member.id)" />
      </span>
    </section>
    <label class="administrator-member-picker__search">
      <RiSearch2Line /><input v-model="keyword" placeholder="搜索（多个关键词用空格隔开）" />
    </label>
    <section class="administrator-member-picker__body">
      <div class="administrator-member-picker__tree">
        <p class="administrator-member-picker__all">全部成员</p>
        <p class="administrator-member-picker__node">
          <RiArrowDownSFill /><RiTeamFill />重庆灵衍云科技有限公司
        </p>
        <p class="administrator-member-picker__child"><RiTeamFill />研发部</p>
        <p class="administrator-member-picker__child"><RiTeamFill />产品部</p>
      </div>
      <div class="administrator-member-picker__results">
        <p class="administrator-member-picker__result-title">
          已选 {{ selectedItems.length }}/{{ members.length }}
        </p>
        <label
          v-for="member in visibleMembers"
          :key="member.id"
          class="administrator-member-picker__member"
        >
          <span class="administrator-member-picker__avatar">{{ member.name.slice(0, 1) }}</span>
          <span>{{ member.name }}</span>
          <el-checkbox
            :model-value="selectedIds.includes(member.id)"
            @change="toggleMember(member.id)"
          />
        </label>
      </div>
    </section>
    <footer class="administrator-member-picker__footer">
      <el-button @click="visible = false">取消</el-button
      ><el-button type="primary" @click="submit">确定</el-button>
    </footer>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.administrator-member-picker) {
  border-radius: 14px;
}
:global(.administrator-member-picker .el-dialog__header) {
  display: none;
}
:global(.administrator-member-picker .el-dialog__body) {
  padding: 0;
}
.administrator-member-picker {
  &__header {
    display: flex;
    height: 68px;
    padding: 0 28px;
    border-bottom: 1px solid #dde2ea;
    align-items: center;
    justify-content: space-between;
  }
  &__header h2 {
    margin: 0;
    color: #273142;
    font-size: 22px;
  }
  &__header button {
    display: inline-flex;
    border: 0;
    padding: 5px;
    color: #66707e;
    background: transparent;
    cursor: pointer;
  }
  &__header button:hover {
    border-radius: 6px;
    background: var(--el-fill-color-light);
  }
  &__header svg {
    width: 25px;
    height: 25px;
  }
  &__selected {
    display: flex;
    min-height: 132px;
    margin: 26px 28px 16px;
    padding: 12px;
    border: 1px dashed #dae0e9;
    border-radius: 9px;
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
    color: #4e5868;
    background: #f4f5f7;
    font-size: 17px;
  }
  &__tag i,
  &__avatar {
    display: inline-flex;
    width: 25px;
    height: 25px;
    border-radius: 50%;
    align-items: center;
    justify-content: center;
    color: #fff;
    background: #f1575e;
    font-size: 14px;
    font-style: normal;
  }
  &__tag svg {
    margin-left: 48px;
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
    background: #f5f6f8;
    color: #697384;
  }
  &__search svg {
    width: 20px;
    height: 20px;
  }
  &__search input {
    width: 100%;
    border: 0;
    outline: 0;
    color: #4e5868;
    background: transparent;
    font: inherit;
  }
  &__body {
    display: grid;
    min-height: 460px;
    margin: 6px 28px 0;
    grid-template-columns: 1fr 1fr;
    border-top: 1px solid #edf0f4;
  }
  &__tree,
  &__results {
    padding: 14px 12px;
  }
  &__tree {
    border-right: 1px solid #e2e6ec;
  }
  &__all {
    margin: 0 0 9px;
    padding: 12px 16px;
    border-radius: 7px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    font-size: 17px;
  }
  &__node,
  &__child {
    display: flex;
    margin: 12px 12px;
    align-items: center;
    gap: 8px;
    color: #4f5968;
    font-size: 17px;
  }
  &__node svg,
  &__child svg {
    color: var(--el-color-primary);
  }
  &__child {
    margin-left: 58px;
  }
  &__result-title {
    margin: 0 0 12px;
    color: var(--el-color-primary);
    font-size: 17px;
  }
  &__member {
    display: flex;
    height: 43px;
    align-items: center;
    gap: 8px;
    color: #4e5868;
    font-size: 17px;
    cursor: pointer;
  }
  &__member .el-checkbox {
    margin-left: auto;
  }
  &__footer {
    display: flex;
    height: 76px;
    padding: 0 28px;
    border-top: 1px solid #dde2ea;
    align-items: center;
    justify-content: flex-end;
    gap: 12px;
  }
  &__footer .el-button {
    min-width: 74px;
    height: 42px;
    font-size: 17px;
  }
}
</style>
