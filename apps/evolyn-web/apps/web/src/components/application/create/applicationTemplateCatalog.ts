import addApplicationContractImage from '~/assets/images/add-app-3.png';
import addApplicationProjectImage from '~/assets/images/add-app-1.png';
import addApplicationTaskImage from '~/assets/images/add-app-2.png';
import templateApplicationImage from '~/assets/images/template-app-1.png';

export interface BlankApplicationStarter {
  id: 'blank';
  title: string;
  image?: undefined;
}

export interface RecommendedApplicationStarter {
  id: 'project' | 'task' | 'contract';
  title: string;
  image: string;
}

export type ApplicationStarter = BlankApplicationStarter | RecommendedApplicationStarter;

export interface ApplicationTemplate {
  id: string;
  title: string;
  description: string;
  image: string;
  imageVariant: 'default' | 'inventory' | 'project' | 'work-order';
}

/**
 * 应用创建入口的临时展示数据。
 * 后续接入模板中心后，只替换此目录的数据来源，弹窗布局与触发方式无需调整。
 */
export const applicationStarters: ApplicationStarter[] = [
  { id: 'blank', title: '创建空白应用' },
  { id: 'project', title: 'IT项目', image: addApplicationProjectImage },
  { id: 'task', title: '任务', image: addApplicationTaskImage },
  { id: 'contract', title: '合同', image: addApplicationContractImage },
];

export const applicationTemplateBatches: ApplicationTemplate[][] = [
  [
    {
      id: 'crm',
      title: 'CRM',
      description: '更强自定义的标准 CRM 客户管理系统',
      image: templateApplicationImage,
      imageVariant: 'default',
    },
    {
      id: 'inventory',
      title: '进销存',
      description: '更强自定义的标准进销存系统',
      image: templateApplicationImage,
      imageVariant: 'inventory',
    },
    {
      id: 'project-management',
      title: '新项目管理',
      description: '通用的项目管理官网应用，包含进度和看板',
      image: templateApplicationImage,
      imageVariant: 'project',
    },
    {
      id: 'work-order',
      title: '生产小工单',
      description: '面向中小微制造业的工厂车间生产管理',
      image: templateApplicationImage,
      imageVariant: 'work-order',
    },
  ],
  [
    {
      id: 'sales-dashboard',
      title: '销售运营',
      description: '聚合销售目标、线索、合同和回款进度',
      image: templateApplicationImage,
      imageVariant: 'inventory',
    },
    {
      id: 'customer-service',
      title: '客户服务',
      description: '统一管理服务工单、客户反馈与处理时效',
      image: templateApplicationImage,
      imageVariant: 'project',
    },
    {
      id: 'purchase-management',
      title: '采购管理',
      description: '覆盖供应商、采购申请和入库协同流程',
      image: templateApplicationImage,
      imageVariant: 'work-order',
    },
    {
      id: 'employee-operations',
      title: '员工运营',
      description: '用一个轻量应用协同人员与日常事项',
      image: templateApplicationImage,
      imageVariant: 'default',
    },
  ],
];
