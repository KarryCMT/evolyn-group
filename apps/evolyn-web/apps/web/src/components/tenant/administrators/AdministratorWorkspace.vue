<script setup lang="ts">
import { shallowRef } from 'vue';
import { useAdministratorManagement } from '~/composables/tenant/useAdministratorManagement';
import AddAdministratorGroupDialog from './AddAdministratorGroupDialog.vue';
import AdministratorGroupList from './AdministratorGroupList.vue';
import AdministratorPermissionPanel from './AdministratorPermissionPanel.vue';
import type { AdministratorGroup, AdministratorScope } from './administrator.types';

defineOptions({ name: 'AdministratorWorkspace' });

const props = defineProps<{ scope: AdministratorScope }>();
const addGroupVisible = shallowRef(false);
const { groups, selectedGroup, selectedId, selectGroup, addGroup } = useAdministratorManagement(
  props.scope,
);

/** 由组合层持有状态，面板只上抛变更，保持列表与详情严格同步。 */
function updateGroup(patch: Partial<AdministratorGroup>) {
  Object.assign(selectedGroup.value, patch);
}
</script>

<template>
  <section class="administrator-workspace">
    <AdministratorGroupList
      :scope="scope"
      :groups="groups"
      :selected-id="selectedId"
      @select="selectGroup"
      @add="addGroupVisible = true"
    />
    <AdministratorPermissionPanel :scope="scope" :group="selectedGroup" @update="updateGroup" />
    <AddAdministratorGroupDialog v-model="addGroupVisible" @confirm="addGroup" />
  </section>
</template>

<style scoped lang="scss">
.administrator-workspace {
  display: flex;
  height: calc(100% - 64px);
  min-height: 0;
  overflow: hidden;
}
</style>
