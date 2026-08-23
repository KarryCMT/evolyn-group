/**
 * 空应用首页的资产创建入口。
 *
 * 这里只定义展示和交互意图；实际创建逻辑会在后续表单/仪表盘包及对应 API
 * 就绪后接入，避免应用首页提前承担资产领域职责。
 */
export type ApplicationAssetType = 'workflow-form' | 'form' | 'dashboard';

export interface ApplicationAssetStarter {
  type: ApplicationAssetType;
  title: string;
  description: string;
  /** app-type-bg.png 中的精灵图位置。 */
  imagePosition: 'workflow-form' | 'form' | 'dashboard';
}

export const applicationAssetStarters: ApplicationAssetStarter[] = [
  {
    type: 'workflow-form',
    title: '新建流程表单',
    description: '适用于有特定流程，需要不同成员分步骤填写数据的业务。',
    imagePosition: 'workflow-form',
  },
  {
    type: 'form',
    title: '新建表单',
    description: '适用于数据上报、问卷调研等业务。',
    imagePosition: 'form',
  },
  {
    type: 'dashboard',
    title: '新建仪表盘',
    description: '可用于分析和展现流程表单、表单搜集到的数据。',
    imagePosition: 'dashboard',
  },
];
