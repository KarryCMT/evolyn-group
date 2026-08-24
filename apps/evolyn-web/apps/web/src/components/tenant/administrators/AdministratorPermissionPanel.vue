<script setup lang="ts">
import { RiAddFill, RiQuestionFill, RiTeamFill, RiUserSettingsFill } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import AddressBookManagementDrawer from './AddressBookManagementDrawer.vue';
import AdministratorApplicationPickerDialog from './AdministratorApplicationPickerDialog.vue';
import AdministratorMemberPickerDialog from './AdministratorMemberPickerDialog.vue';
import {
  administratorApplications,
  administratorMembers,
  departments,
  roles,
  type AdministratorGroup,
  type AdministratorScope,
  type ScopeMode,
} from './administrator.types';

defineOptions({ name: 'AdministratorPermissionPanel' });

const props = defineProps<{ scope: AdministratorScope; group: AdministratorGroup }>();
const emit = defineEmits<{ update: [patch: Partial<AdministratorGroup>] }>();
const memberPickerVisible = shallowRef(false);
const applicationPickerVisible = shallowRef(false);
const addressBookVisible = shallowRef(false);
const showSaved = shallowRef(false);
const applications = computed(() =>
  administratorApplications.filter((application) =>
    props.group.applicationIds.includes(application.id),
  ),
);

function patch(patchValue: Partial<AdministratorGroup>) {
  emit('update', patchValue);
  showSaved.value = true;
  window.setTimeout(() => {
    showSaved.value = false;
  }, 1400);
}

function setDepartmentMode(mode: ScopeMode) {
  patch({ departmentMode: mode });
}
function setRoleMode(mode: ScopeMode) {
  patch({ roleMode: mode });
}
</script>

<template>
  <section class="administrator-permission-panel" aria-label="管理组权限">
    <header class="administrator-permission-panel__header">
      <div>
        <h1>{{ group.name }}</h1>
        <p v-if="scope === 'system' && group.builtIn">
          系统管理员具备全产品/模块的全量管理及数据权限，建议配置不超过 5 人
        </p>
      </div>
    </header>
    <div class="administrator-permission-panel__body">
      <section
        class="administrator-permission-panel__row administrator-permission-panel__row--members"
      >
        <h2>管理员</h2>
        <button
          class="administrator-permission-panel__text-action"
          type="button"
          @click="memberPickerVisible = true"
        >
          <RiAddFill />选择成员
        </button>
        <span
          v-for="member in group.members"
          :key="member.id"
          class="administrator-permission-panel__member"
          ><i>{{ member.name.slice(0, 1) }}</i
          >{{ member.name }}</span
        >
      </section>

      <template v-if="scope === 'system'">
        <section class="administrator-permission-panel__row">
          <h2>内部部门</h2>
          <el-checkbox
            :model-value="group.departmentEnabled"
            @update:model-value="patch({ departmentEnabled: Boolean($event) })"
            >可见/可管理</el-checkbox
          >
        </section>
        <section
          v-if="group.departmentEnabled"
          class="administrator-permission-panel__scope-detail"
        >
          <el-radio-group
            :model-value="group.departmentMode"
            @update:model-value="setDepartmentMode($event as ScopeMode)"
            ><el-radio value="all">全部部门</el-radio
            ><el-radio value="partial">部分部门</el-radio></el-radio-group
          >
          <div
            v-if="group.departmentMode === 'partial'"
            class="administrator-permission-panel__selection-box"
          >
            <span
              v-for="department in departments.filter((item) =>
                group.departmentIds.includes(item.id),
              )"
              :key="department.id"
              ><RiTeamFill />{{ department.name }}</span
            >
          </div>
        </section>
        <section class="administrator-permission-panel__row">
          <h2>内部角色</h2>
          <div class="administrator-permission-panel__checkbox-pair">
            <el-checkbox
              :model-value="group.roleVisible"
              @update:model-value="patch({ roleVisible: Boolean($event) })"
              >可见</el-checkbox
            ><el-checkbox
              :model-value="group.roleManage"
              @update:model-value="patch({ roleManage: Boolean($event) })"
              >可管理</el-checkbox
            >
          </div>
        </section>
        <section
          v-if="group.roleVisible || group.roleManage"
          class="administrator-permission-panel__scope-detail"
        >
          <el-radio-group
            :model-value="group.roleMode"
            @update:model-value="setRoleMode($event as ScopeMode)"
            ><el-radio value="all">全部角色</el-radio
            ><el-radio value="partial">部分角色</el-radio></el-radio-group
          >
          <div
            v-if="group.roleMode === 'partial'"
            class="administrator-permission-panel__selection-box"
          >
            <span
              v-for="role in roles.filter((item) => group.roleIds.includes(item.id))"
              :key="role.id"
              ><RiUserSettingsFill />{{ role.name }}</span
            >
          </div>
        </section>
        <section class="administrator-permission-panel__row">
          <h2>互联组织</h2>
          <el-checkbox
            :model-value="group.externalEnabled"
            @update:model-value="patch({ externalEnabled: Boolean($event) })"
            >可见/可管理 <RiQuestionFill
          /></el-checkbox>
        </section>
      </template>

      <template v-else>
        <section class="administrator-permission-panel__row">
          <h2>应用管理</h2>
          <button
            class="administrator-permission-panel__text-action"
            type="button"
            @click="applicationPickerVisible = true"
          >
            <RiAddFill />选择可编辑的应用
          </button>
        </section>
        <section class="administrator-permission-panel__app-settings">
          <el-checkbox
            :model-value="group.applicationManage"
            @update:model-value="patch({ applicationManage: Boolean($event) })"
            >可添加/删除应用</el-checkbox
          >
          <div class="administrator-permission-panel__app-tags">
            <span v-for="application in applications" :key="application.id"
              ><i :class="`administrator-permission-panel__app-icon--${application.tone}`">{{
                application.icon
              }}</i
              >{{ application.name }}</span
            >
          </div>
        </section>
        <section class="administrator-permission-panel__row">
          <h2>可选部门 <RiQuestionFill /></h2>
          <el-radio-group
            :model-value="group.departmentMode"
            @update:model-value="setDepartmentMode($event as ScopeMode)"
            ><el-radio value="all">全部部门</el-radio
            ><el-radio value="partial">部分部门</el-radio></el-radio-group
          >
        </section>
        <section class="administrator-permission-panel__sub-action">
          <button class="administrator-permission-panel__text-action" type="button">
            <RiAddFill />选择部门
          </button>
        </section>
        <section class="administrator-permission-panel__row">
          <h2>可选角色 <RiQuestionFill /></h2>
          <el-radio-group
            :model-value="group.roleMode"
            @update:model-value="setRoleMode($event as ScopeMode)"
            ><el-radio value="all">全部角色</el-radio
            ><el-radio value="partial">部分角色</el-radio></el-radio-group
          >
        </section>
        <section class="administrator-permission-panel__sub-action">
          <button class="administrator-permission-panel__text-action" type="button">
            <RiAddFill />选择角色
          </button>
        </section>
        <section class="administrator-permission-panel__row">
          <h2>可选互联组织 <RiQuestionFill /></h2>
          <el-checkbox
            :model-value="group.externalEnabled"
            @update:model-value="patch({ externalEnabled: Boolean($event) })"
            >全部互联组织</el-checkbox
          >
        </section>
        <section
          class="administrator-permission-panel__row administrator-permission-panel__row--address-book"
        >
          <h2>通讯录管理</h2>
          <div>
            <p>
              可为成员设置通讯录管理权限，并可在通讯录管理组中查看。
              <button type="button">了解更多</button>
            </p>
            <el-button plain type="primary" @click="addressBookVisible = true">设置</el-button>
          </div>
        </section>
      </template>
      <transition name="administrator-saved"
        ><span v-if="showSaved" class="administrator-permission-panel__saved"
          >✓ 修改已保存</span
        ></transition
      >
    </div>
    <AdministratorMemberPickerDialog
      v-model="memberPickerVisible"
      :members="administratorMembers"
      :selected-members="group.members"
      @confirm="patch({ members: $event })"
    />
    <AdministratorApplicationPickerDialog
      v-if="scope === 'application'"
      v-model="applicationPickerVisible"
      :applications="administratorApplications"
      :selected-ids="group.applicationIds"
      @confirm="patch({ applicationIds: $event })"
    />
    <AddressBookManagementDrawer
      v-if="scope === 'application'"
      v-model="addressBookVisible"
      :group="group"
    />
  </section>
</template>

<style scoped lang="scss">
.administrator-permission-panel {
  display: flex;
  min-width: 0;
  height: 100%;
  flex: 1;
  flex-direction: column;
  background: #fff;
  &__header {
    display: flex;
    min-height: 68px;
    padding: 0 30px;
    border-bottom: 1px solid #e5e8ee;
    align-items: center;
  }
  &__header div {
    display: flex;
    align-items: baseline;
    gap: 14px;
  }
  &__header h1 {
    margin: 0;
    color: #253042;
    font-size: 21px;
    line-height: 28px;
  }
  &__header p {
    margin: 0;
    color: #7d8796;
    font-size: 17px;
  }
  &__body {
    position: relative;
    padding: 26px 30px 80px;
  }
  &__row {
    display: grid;
    min-height: 64px;
    grid-template-columns: 246px 1fr;
    align-items: start;
  }
  &__row h2 {
    display: inline-flex;
    margin: 0;
    align-items: center;
    gap: 5px;
    color: #5d6675;
    font-size: 18px;
    line-height: 30px;
  }
  &__row h2 svg,
  &__row :deep(.el-checkbox svg) {
    width: 18px;
    height: 18px;
    color: #aeb6c3;
  }
  &__row :deep(.el-checkbox),
  &__row :deep(.el-radio) {
    min-height: 30px;
    margin-right: 28px;
    color: #465061;
    font-size: 17px;
  }
  &__row :deep(.el-checkbox__label),
  &__row :deep(.el-radio__label) {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 17px;
  }
  &__row--members {
    align-items: center;
  }
  &__text-action {
    display: inline-flex;
    min-height: 32px;
    border: 0;
    padding: 0;
    align-items: center;
    gap: 5px;
    color: var(--el-color-primary);
    background: transparent;
    font-size: 17px;
    cursor: pointer;
  }
  &__text-action svg {
    width: 22px;
    height: 22px;
  }
  &__text-action:hover {
    text-decoration: underline;
  }
  &__member {
    display: inline-flex;
    height: 32px;
    margin-left: 12px;
    align-items: center;
    gap: 6px;
    color: #4d5765;
    font-size: 16px;
  }
  &__member i {
    display: inline-flex;
    width: 24px;
    height: 24px;
    border-radius: 50%;
    align-items: center;
    justify-content: center;
    color: #fff;
    background: #f1575e;
    font-size: 13px;
    font-style: normal;
  }
  &__scope-detail {
    margin: -5px 0 28px 246px;
  }
  &__scope-detail :deep(.el-radio) {
    color: #465061;
    font-size: 17px;
  }
  &__selection-box {
    display: flex;
    min-height: 92px;
    margin-top: 18px;
    padding: 12px;
    border: 1px dashed #d8dee8;
    border-radius: 7px;
    align-content: flex-start;
    align-items: flex-start;
    flex-wrap: wrap;
    gap: 8px;
  }
  &__selection-box span {
    display: inline-flex;
    height: 34px;
    padding: 0 10px;
    border-radius: 5px;
    align-items: center;
    gap: 5px;
    color: #4d5666;
    background: #f4f5f7;
    font-size: 16px;
  }
  &__selection-box svg {
    color: var(--el-color-primary);
  }
  &__checkbox-pair {
    display: flex;
    align-items: center;
  }
  &__app-settings {
    display: flex;
    min-height: 90px;
    margin: -6px 0 8px 246px;
    flex-direction: column;
    gap: 14px;
  }
  &__app-settings :deep(.el-checkbox) {
    color: #465061;
    font-size: 17px;
  }
  &__app-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  &__app-tags span {
    display: inline-flex;
    height: 34px;
    padding: 0 10px;
    border-radius: 5px;
    align-items: center;
    gap: 5px;
    color: #4d5666;
    background: #f4f5f7;
    font-size: 15px;
  }
  &__app-icon--green,
  &__app-icon--coral,
  &__app-icon--blue,
  &__app-icon--cyan,
  &__app-icon--purple,
  &__app-icon--orange {
    display: inline-flex;
    width: 20px;
    height: 20px;
    border-radius: 4px;
    align-items: center;
    justify-content: center;
    color: #fff;
    font-size: 9px;
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
  &__sub-action {
    margin: -18px 0 28px 246px;
  }
  &__row--address-book {
    margin-top: 12px;
  }
  &__row--address-book p {
    margin: 0 0 16px;
    color: #a0a8b5;
    font-size: 16px;
  }
  &__row--address-book p button {
    border: 0;
    padding: 0;
    color: var(--el-color-primary);
    background: transparent;
    font: inherit;
    cursor: pointer;
  }
  &__row--address-book p button:hover {
    text-decoration: underline;
  }
  &__row--address-book .el-button {
    min-width: 72px;
    height: 42px;
    font-size: 17px;
  }
  &__saved {
    position: absolute;
    top: 365px;
    right: 30px;
    color: var(--el-color-success);
    font-size: 16px;
  }
}
.administrator-saved-enter-active,
.administrator-saved-leave-active {
  transition: opacity 0.2s;
}
.administrator-saved-enter-from,
.administrator-saved-leave-to {
  opacity: 0;
}
</style>
