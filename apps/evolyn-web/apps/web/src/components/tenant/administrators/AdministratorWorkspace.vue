<script setup lang="ts">
import { onMounted, shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessageBox } from 'element-plus';
import AddAdministratorGroupDialog from './AddAdministratorGroupDialog.vue';
import AdministratorGroupList from './AdministratorGroupList.vue';
import AdministratorPermissionPanel from './AdministratorPermissionPanel.vue';
import type { AdministratorGroup, AdministratorScope } from './administrator.types';
import { listMembers } from '~/api/member';
import { useAdministratorManagement } from '~/composables/tenant/useAdministratorManagement';

defineOptions({ name: 'AdministratorWorkspace' });

const props = defineProps<{ scope: AdministratorScope }>();
const router = useRouter();
const addGroupVisible = shallowRef(false);
const {
  groups,
  selectedGroup,
  selectedId,
  listLoading,
  detailLoading,
  saving,
  loadGroups,
  selectGroup,
  addGroup,
  applyPatch,
} = useAdministratorManagement(props.scope);

onMounted(() => {
  void loadGroups().then(checkInvitePrompt);
});

/**
 * 租户内仅有创建人一名成员时提示邀请成员（MessageBox 二次确认形态）：
 * 在职成员总数 ≤1（创建人恒为成员，唯一成员即创建人）即提示，且每次进入
 * 管理员页都会重复提示，直到邀请成员（总数 >1）为止。确认（立即邀请）
 * 前往内部组织页；取消或按 Esc（暂不邀请）仅关闭，不记忆。判定失败静默
 * 跳过——提示属于增强体验，不阻塞页面。
 */
async function checkInvitePrompt() {
  try {
    const page = await listMembers({ page: 1, pageSize: 1 });
    if (page.total > 1) return;
  } catch {
    return;
  }

  try {
    await ElMessageBox.confirm(
      '邀请的成员加入企业后，可以填报数据、处理流程等；你也可以将他/她设置为管理员，协助管理应用。',
      '企业中没有成员，是否邀请？',
      {
        type: 'warning',
        confirmButtonText: '立即邀请',
        cancelButtonText: '暂不邀请',
        showClose: false,
        closeOnClickModal: false,
        customStyle: { width: '520px', borderRadius: '14px' },
      },
    );
    void router.push({ name: 'tenant-organization' });
  } catch {
    // 暂不邀请：仅关闭，下次进入管理员页仍会提示
  }
}

/**
 * 面板上抛的区块变更经组合层落库：乐观更新 + 失败回滚都在 applyPatch 内，
 * 返回值告知面板是否展示「修改已保存」。列表与详情由此保持严格同步。
 */
function updateGroup(patch: Partial<AdministratorGroup>): Promise<boolean> {
  return applyPatch(patch);
}
</script>

<template>
  <section class="administrator-workspace">
    <AdministratorGroupList
      :scope="scope"
      :groups="groups"
      :selected-id="selectedId"
      :loading="listLoading"
      @select="selectGroup"
      @add="addGroupVisible = true"
    />
    <AdministratorPermissionPanel
      :scope="scope"
      :group="selectedGroup"
      :loading="detailLoading"
      :saving="saving"
      :save="updateGroup"
    />
    <AddAdministratorGroupDialog v-model="addGroupVisible" :submit="addGroup" />
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
