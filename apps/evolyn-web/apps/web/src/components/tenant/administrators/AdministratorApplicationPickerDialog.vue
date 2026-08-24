<script setup lang="ts">
import { RiCloseFill, RiSearch2Line } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import type { AdministratorApplication } from './administrator.types';

defineOptions({ name: 'AdministratorApplicationPickerDialog' });

const props = defineProps<{ applications: AdministratorApplication[]; selectedIds: string[] }>();
const visible = defineModel<boolean>({ default: false });
const emit = defineEmits<{ confirm: [ids: string[]] }>();
const keyword = shallowRef('');
const draftIds = shallowRef<string[]>([]);
const filteredApplications = computed(() => {
  const query = keyword.value.trim();
  return query ? props.applications.filter((app) => app.name.includes(query)) : props.applications;
});
const selectedApps = computed(() =>
  props.applications.filter((app) => draftIds.value.includes(app.id)),
);
const allSelected = computed(() => draftIds.value.length === props.applications.length);

function toggleAll(value: string | number | boolean) {
  draftIds.value = value === true ? props.applications.map((app) => app.id) : [];
}
function submit() {
  emit('confirm', draftIds.value);
  visible.value = false;
}
function remove(id: string) {
  draftIds.value = draftIds.value.filter((item) => item !== id);
}
watch(visible, (isVisible) => {
  if (isVisible) {
    keyword.value = '';
    draftIds.value = [...props.selectedIds];
  }
});
</script>

<template>
  <el-dialog
    v-model="visible"
    class="administrator-application-picker"
    width="944px"
    :show-close="false"
    append-to-body
  >
    <header class="administrator-application-picker__header">
      <h2>应用列表</h2>
      <button type="button" aria-label="关闭" @click="visible = false"><RiCloseFill /></button>
    </header>
    <section class="administrator-application-picker__body">
      <div class="administrator-application-picker__catalog">
        <label class="administrator-application-picker__search"
          ><RiSearch2Line /><input v-model="keyword" placeholder="搜索"
        /></label>
        <el-checkbox
          :model-value="allSelected"
          :indeterminate="draftIds.length > 0 && !allSelected"
          @change="toggleAll"
          >全选</el-checkbox
        >
        <el-checkbox-group v-model="draftIds" class="administrator-application-picker__list">
          <el-checkbox v-for="app in filteredApplications" :key="app.id" :value="app.id"
            ><i :class="`administrator-application-picker__app-icon--${app.tone}`">{{ app.icon }}</i
            >{{ app.name }}</el-checkbox
          >
        </el-checkbox-group>
      </div>
      <div class="administrator-application-picker__selected">
        <span
          v-for="app in selectedApps"
          :key="app.id"
          class="administrator-application-picker__tag"
          ><i :class="`administrator-application-picker__app-icon--${app.tone}`">{{ app.icon }}</i
          >{{ app.name }}<RiCloseFill @click="remove(app.id)"
        /></span>
      </div>
    </section>
    <footer class="administrator-application-picker__footer">
      <el-button @click="visible = false">取消</el-button
      ><el-button type="primary" @click="submit">确定</el-button>
    </footer>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.administrator-application-picker) {
  border-radius: 14px;
}
:global(.administrator-application-picker .el-dialog__header) {
  display: none;
}
:global(.administrator-application-picker .el-dialog__body) {
  padding: 0;
}
.administrator-application-picker {
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
  &__body {
    display: grid;
    height: 652px;
    grid-template-columns: 1fr 1fr;
    margin: 24px 28px 16px;
    border: 1px solid #dce2ea;
    border-radius: 8px;
    overflow: hidden;
  }
  &__catalog {
    display: flex;
    flex-direction: column;
    padding: 16px 20px;
    border-right: 1px solid #dde2ea;
    gap: 12px;
  }
  &__search {
    display: flex;
    height: 42px;
    padding: 0 12px;
    border-radius: 7px;
    align-items: center;
    gap: 8px;
    color: #687383;
    background: #f5f6f8;
  }
  &__search svg {
    width: 20px;
    height: 20px;
  }
  &__search input {
    width: 100%;
    border: 0;
    outline: 0;
    background: transparent;
    font: inherit;
  }
  &__list {
    display: flex;
    flex-direction: column;
    gap: 13px;
  }
  &__list :deep(.el-checkbox) {
    height: 27px;
    margin-right: 0;
  }
  &__list :deep(.el-checkbox__label) {
    display: inline-flex;
    align-items: center;
    gap: 12px;
    color: #4c5666;
    font-size: 17px;
  }
  &__app-icon--green,
  &__app-icon--coral,
  &__app-icon--blue,
  &__app-icon--cyan,
  &__app-icon--purple,
  &__app-icon--orange {
    display: inline-flex;
    width: 28px;
    height: 28px;
    border-radius: 5px;
    align-items: center;
    justify-content: center;
    color: #fff;
    font-size: 12px;
    font-style: normal;
    font-weight: 700;
  }
  &__app-icon--green {
    background: #5bcf73;
  }
  &__app-icon--coral {
    background: #f56b70;
  }
  &__app-icon--blue {
    background: #5b91f5;
  }
  &__app-icon--cyan {
    background: #2eb3d7;
  }
  &__app-icon--purple {
    background: #8565ec;
  }
  &__app-icon--orange {
    background: #f5ad35;
  }
  &__selected {
    display: flex;
    padding: 16px 20px;
    align-content: flex-start;
    align-items: flex-start;
    flex-wrap: wrap;
    gap: 8px;
  }
  &__tag {
    display: inline-flex;
    height: 42px;
    gap: 8px;
    padding: 0 10px;
    border-radius: 7px;
    align-items: center;
    color: #4d5766;
    background: #f4f5f7;
    font-size: 16px;
  }
  &__tag svg {
    margin-left: 4px;
    color: #6d7785;
    cursor: pointer;
  }
  &__tag svg:hover {
    color: var(--el-color-danger);
  }
  &__footer {
    display: flex;
    height: 76px;
    padding: 0 28px;
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
