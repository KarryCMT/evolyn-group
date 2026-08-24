import { computed, type Ref } from 'vue';
import {
  RiCalendarScheduleFill,
  RiCheckboxMultipleFill,
  RiHashtag,
  RiOrganizationChart,
  RiTableFill,
  RiText,
  RiUser3Fill,
} from '@remixicon/vue';
import type { PluginFieldMetadata } from '../api';
import type { PluginDesignPaletteItem } from '../types';

type PluginDesignPaletteConfig = Omit<PluginDesignPaletteItem, 'label' | 'icon'>;

// 字段项直接携带函数接口所需的组件属性，点击和拖拽新增时原样透传。
const commonPaletteConfigs: Array<PluginDesignPaletteConfig & { metadataType: string }> = [
  { metadataType: 'text', widgetName: 'input', dataType: 'String' },
  { metadataType: 'number', widgetName: 'number', dataType: 'Number' },
  { metadataType: 'datetime', widgetName: 'datetime', dataType: 'Date' },
  { metadataType: 'select', widgetName: 'selectGroup', dataType: 'Object' },
];

const requestPaletteConfigs: Array<PluginDesignPaletteConfig & { metadataType: string }> = [
  ...commonPaletteConfigs,
  { metadataType: 'user', widgetName: 'userGroup', dataType: 'Object' },
  { metadataType: 'dept', widgetName: 'deptGroup', dataType: 'Object' },
  { metadataType: 'vector', widgetName: 'subforms', dataType: '' },
];

// 字段入口统一使用 Remix Icon 中实际可用的图标，并按字段数据语义区分展示。
const fieldIconMap = {
  input: RiText,
  number: RiHashtag,
  datetime: RiCalendarScheduleFill,
  selectGroup: RiCheckboxMultipleFill,
  userGroup: RiUser3Fill,
  deptGroup: RiOrganizationChart,
  subforms: RiTableFill,
};

export const useFormDesignPalette = (fieldMetadataList: Ref<PluginFieldMetadata[]>) => {
  const widgetNameTextMap = computed<Record<string, string>>(() => ({
    input: '文本',
    number: '数字',
    datetime: '日期时间',
    selectGroup: '下拉框',
    userGroup: '成员选择',
    deptGroup: '部门选择',
    subforms: '子表单',
  }));

  const createPaletteItemFromMetadata = (
    config: PluginDesignPaletteConfig & { metadataType: string },
  ): PluginDesignPaletteItem | null => {
    const metadata = fieldMetadataList.value.find(
      (item) => item.field_type === config.metadataType,
    );
    if (!metadata) return null;
    return {
      label: widgetNameTextMap.value[config.widgetName] || config.widgetName,
      widgetName: config.widgetName,
      dataType: config.dataType,
      icon: fieldIconMap[config.widgetName as keyof typeof fieldIconMap] || RiText,
    };
  };

  const createPaletteByMetadata = (
    configs: Array<PluginDesignPaletteConfig & { metadataType: string }>,
  ) => {
    // 字段按钮从字段元数据生成，元数据缺失时不展示对应入口。
    return configs
      .map(createPaletteItemFromMetadata)
      .filter((item): item is PluginDesignPaletteItem => Boolean(item));
  };

  const commonPalette = computed(() => createPaletteByMetadata(commonPaletteConfigs));
  const requestPalette = computed(() => createPaletteByMetadata(requestPaletteConfigs));

  return {
    commonPalette,
    requestPalette,
    widgetNameTextMap,
  };
};
