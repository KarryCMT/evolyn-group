import type { CrossAppForm, CrossAppFormGroup, CrossAppSourceApp } from './crossApp.types';
import { computed, shallowRef } from 'vue';
import { crossAppDemoApps, initialCrossAppFormIDs } from './crossAppDemo';

export interface SelectedCrossAppForm extends CrossAppForm {
  appId: string;
  appName: string;
}

function containsKeyword(value: string, keyword: string) {
  return value.toLocaleLowerCase().includes(keyword.toLocaleLowerCase());
}

/**
 * 跨应用页面的本地交互状态。选择结果在本 composable 中集中维护，展示组件
 * 只通过 props 接收数据、通过 emits 上抛动作，后续接 API 时无需改动子组件。
 */
export function useCrossAppSelection() {
  const keyword = shallowRef('');
  const activeAppId = shallowRef(crossAppDemoApps[0].id);
  const selectedFormIds = shallowRef<string[]>([...initialCrossAppFormIDs]);
  const savedFormIds = shallowRef<string[]>([...initialCrossAppFormIDs]);

  const allForms = computed<SelectedCrossAppForm[]>(() =>
    crossAppDemoApps.flatMap((app) =>
      app.groups.flatMap((group) =>
        group.forms.map((form) => ({
          ...form,
          appId: app.id,
          appName: app.name,
        })),
      ),
    ),
  );
  const selectedIdSet = computed(() => new Set(selectedFormIds.value));
  const selectedForms = computed(() =>
    allForms.value.filter((form) => selectedIdSet.value.has(form.id)),
  );
  const hasUnsavedChanges = computed(() => {
    const saved = new Set(savedFormIds.value);
    return (
      saved.size !== selectedIdSet.value.size || selectedFormIds.value.some((id) => !saved.has(id))
    );
  });

  const filteredApps = computed<CrossAppSourceApp[]>(() => {
    const normalizedKeyword = keyword.value.trim();
    if (!normalizedKeyword) return crossAppDemoApps;

    return crossAppDemoApps
      .map((app) => {
        if (containsKeyword(app.name, normalizedKeyword)) return app;
        const groups = app.groups
          .map((group) => {
            if (containsKeyword(group.name, normalizedKeyword)) return group;
            const forms = group.forms.filter((form) =>
              containsKeyword(form.name, normalizedKeyword),
            );
            return forms.length ? { ...group, forms } : null;
          })
          .filter((group): group is CrossAppFormGroup => group !== null);
        return groups.length ? { ...app, groups } : null;
      })
      .filter((app): app is CrossAppSourceApp => app !== null);
  });

  // 搜索将当前应用排除时，展示列表首项；保留主动选择，清空搜索后可回到原来源。
  const activeApp = computed(
    () =>
      filteredApps.value.find((app) => app.id === activeAppId.value) ??
      filteredApps.value[0] ??
      null,
  );

  function selectApp(id: string) {
    activeAppId.value = id;
  }

  function isFormSelected(id: string) {
    return selectedIdSet.value.has(id);
  }

  function toggleForm(id: string) {
    selectedFormIds.value = isFormSelected(id)
      ? selectedFormIds.value.filter((selectedId) => selectedId !== id)
      : [...selectedFormIds.value, id];
  }

  function toggleGroup(formIds: string[]) {
    const allSelected = formIds.every((id) => selectedIdSet.value.has(id));
    selectedFormIds.value = allSelected
      ? selectedFormIds.value.filter((id) => !formIds.includes(id))
      : [...new Set([...selectedFormIds.value, ...formIds])];
  }

  function removeSelectedForm(id: string) {
    selectedFormIds.value = selectedFormIds.value.filter((selectedId) => selectedId !== id);
  }

  /** 当前仅保存本地演示态；接口接入后在此替换为提交并用服务端结果覆盖。 */
  function save() {
    savedFormIds.value = [...selectedFormIds.value];
  }

  return {
    keyword,
    activeApp,
    activeAppId,
    filteredApps,
    hasUnsavedChanges,
    selectedFormIds,
    selectedForms,
    isFormSelected,
    removeSelectedForm,
    save,
    selectApp,
    toggleForm,
    toggleGroup,
  };
}
