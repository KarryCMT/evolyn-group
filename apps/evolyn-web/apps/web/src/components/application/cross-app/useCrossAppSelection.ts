import type { CrossAppForm, CrossAppFormGroup, CrossAppSourceApplication } from './crossApp.types';
import { computed, shallowRef } from 'vue';
import { crossAppDemoApplications, initialCrossAppFormIDs } from './crossAppDemo';

export interface SelectedCrossAppForm extends CrossAppForm {
  applicationId: string;
  applicationName: string;
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
  const activeApplicationId = shallowRef(crossAppDemoApplications[0].id);
  const selectedFormIds = shallowRef<string[]>([...initialCrossAppFormIDs]);
  const savedFormIds = shallowRef<string[]>([...initialCrossAppFormIDs]);

  const allForms = computed<SelectedCrossAppForm[]>(() =>
    crossAppDemoApplications.flatMap((application) =>
      application.groups.flatMap((group) =>
        group.forms.map((form) => ({
          ...form,
          applicationId: application.id,
          applicationName: application.name,
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

  const filteredApplications = computed<CrossAppSourceApplication[]>(() => {
    const normalizedKeyword = keyword.value.trim();
    if (!normalizedKeyword) return crossAppDemoApplications;

    return crossAppDemoApplications
      .map((application) => {
        if (containsKeyword(application.name, normalizedKeyword)) return application;
        const groups = application.groups
          .map((group) => {
            if (containsKeyword(group.name, normalizedKeyword)) return group;
            const forms = group.forms.filter((form) =>
              containsKeyword(form.name, normalizedKeyword),
            );
            return forms.length ? { ...group, forms } : null;
          })
          .filter((group): group is CrossAppFormGroup => group !== null);
        return groups.length ? { ...application, groups } : null;
      })
      .filter((application): application is CrossAppSourceApplication => application !== null);
  });

  // 搜索将当前应用排除时，展示列表首项；保留主动选择，清空搜索后可回到原来源。
  const activeApplication = computed(
    () =>
      filteredApplications.value.find(
        (application) => application.id === activeApplicationId.value,
      ) ??
      filteredApplications.value[0] ??
      null,
  );

  function selectApplication(id: string) {
    activeApplicationId.value = id;
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
    activeApplication,
    activeApplicationId,
    filteredApplications,
    hasUnsavedChanges,
    selectedFormIds,
    selectedForms,
    isFormSelected,
    removeSelectedForm,
    save,
    selectApplication,
    toggleForm,
    toggleGroup,
  };
}
