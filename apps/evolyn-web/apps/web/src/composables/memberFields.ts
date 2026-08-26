import { storeToRefs } from 'pinia';
import { readonly } from 'vue';
import { useMemberFieldsStore } from '~/stores/memberFields';

// 成员字段配置消费入口：memberFields store（stores/memberFields.ts）之上的
// 只读适配层——快照/加载态以 readonly 暴露，字段开关的乐观更新直接改行属性、
// 提交/回滚一律走 store 提供的 updateField（消费方：字段设置页、卡片展示页）
export function useMemberFields() {
  const store = useMemberFieldsStore();
  const { snapshot, loading, fields, revision, loaded } = storeToRefs(store);

  return {
    snapshot: readonly(snapshot),
    loading,
    fields,
    revision,
    loaded,
    load: store.load,
    updateField: store.updateField,
  };
}
