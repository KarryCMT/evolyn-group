import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import {
  getMemberFieldSettings,
  updateMemberFieldSetting,
  type MemberFieldConfigSnapshotDto,
  type MemberFieldSettingDto,
} from '~/api/memberField';

// 成员字段配置 store（成员信息管理一期）：字段设置页与卡片展示页签共用的
// 服务端配置唯一前端镜像——两页签切换不重复拉取/维护配置副本。勾选采用
// 乐观更新：页面先改行内开关并暂存旧值，再经 updateField 提交；成功后以
// 服务端响应整页覆盖本地快照（含新 revision），失败时由页面按暂存旧值
// 回滚控件状态并按 errCode 提示。消费方统一走 composables/memberFields.ts
export const useMemberFieldsStore = defineStore('memberFields', () => {
  // 整体替换的拉取型数据：快照对象由服务端单点生成；行内开关的乐观更新
  // 由页面直接改行属性（数组元素为响应式对象），提交后仍以响应整页对齐
  const snapshot = ref<MemberFieldConfigSnapshotDto | null>(null);
  const loading = ref(false);

  const fields = computed<MemberFieldSettingDto[]>(() => snapshot.value?.fields ?? []);
  const revision = computed(() => snapshot.value?.revision ?? 0);
  const loaded = computed(() => snapshot.value !== null);

  /** 首屏拉取配置快照；已加载时静默复用（页签切换不重复请求），force 强制刷新。 */
  async function load(force = false): Promise<MemberFieldConfigSnapshotDto | null> {
    if (loaded.value && !force) return snapshot.value;
    loading.value = true;
    try {
      snapshot.value = await getMemberFieldSettings();
      return snapshot.value;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 单字段即时保存：提交本次变更的开关与页面版本号；成功后以服务端最新
   * 快照整页覆盖（含新 revision）。失败抛出原始错误（errCode 分支见
   * packages/utils 的 errorCodes），由调用方回滚控件状态并提示。
   */
  async function updateField(
    fieldKey: string,
    changes: Partial<
      Pick<MemberFieldSettingDto, 'personalVisible' | 'personalEditable' | 'cardVisible'>
    >,
  ): Promise<MemberFieldConfigSnapshotDto> {
    const latest = await updateMemberFieldSetting(fieldKey, {
      ...changes,
      revision: revision.value,
    });
    snapshot.value = latest;
    return latest;
  }

  return { snapshot, loading, fields, revision, loaded, load, updateField };
});
