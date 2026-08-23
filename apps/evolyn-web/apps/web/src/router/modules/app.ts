import type { RouteRecordRaw } from 'vue-router';

interface AppSettingFeatureProps {
  title: string;
  description: string;
  capabilities: string[];
}

/** 规划阶段的应用后台功能页共用展示壳；后续按业务模块替换为实际页面。 */
function settingFeature(props: AppSettingFeatureProps) {
  return {
    component: () => import('~/pages/app/setting-feature.vue'),
    props,
  };
}

/**
 * 应用路由集合
 */
const appRoutes: RouteRecordRaw[] = [
  {
    path: '/app/:appCode',
    name: 'App',
    component: () => import('~/pages/app/index.vue'),
    meta: { public: false },
  },
  {
    path: '/app/:appCode/setting',
    name: 'app-setting',
    component: () => import('~/pages/app/setting.vue'),
    meta: { public: false },
    redirect: { name: 'app-setting-permissions' },
    children: [
      {
        path: 'permissions',
        name: 'app-setting-permissions',
        ...settingFeature({
          title: '表单/仪表盘权限',
          description: '选择表单或仪表盘后，为成员配置查看、填写和管理权限。',
          capabilities: ['表单与仪表盘选择', '成员及部门授权', '权限组启停与调整'],
        }),
      },
      {
        path: 'cross-app',
        name: 'app-setting-cross-app',
        ...settingFeature({
          title: '跨应用',
          description: '管理当前应用与其他应用之间的数据访问和协作关系。',
          capabilities: ['跨应用数据关联', '访问范围控制', '协作关系管理'],
        }),
      },
      {
        path: 'basic',
        name: 'app-setting-basic',
        ...settingFeature({
          title: '应用设置',
          description: '维护应用名称、图标、访问策略与基础运行配置。',
          capabilities: ['应用基本信息', '访问与分享设置', '应用归档与恢复'],
        }),
      },
      {
        path: 'management-groups',
        name: 'app-setting-management-groups',
        ...settingFeature({
          title: '应用管理组',
          description: '将应用管理职责分配给指定成员或部门。',
          capabilities: ['管理组成员维护', '管理范围配置', '管理操作审计'],
        }),
      },
      {
        path: 'aggregate-tables',
        name: 'app-setting-aggregate-tables',
        ...settingFeature({
          title: '聚合表',
          description: '配置跨表数据汇总与面向业务的聚合视图。',
          capabilities: ['聚合规则管理', '字段汇总配置', '聚合结果预览'],
        }),
      },
      {
        path: 'calculations',
        name: 'app-setting-calculations',
        ...settingFeature({
          title: '计算',
          description: '统一管理应用内的计算规则和自动处理逻辑。',
          capabilities: ['计算规则配置', '执行状态追踪', '异常处理记录'],
        }),
      },
      {
        path: 'ai-assistant',
        name: 'app-setting-ai-assistant',
        ...settingFeature({
          title: '智能助手',
          description: '配置应用可用的智能能力及其访问范围。',
          capabilities: ['智能能力开关', '成员使用范围', '调用记录查看'],
        }),
      },
      {
        path: 'data-factory',
        name: 'app-setting-data-factory',
        ...settingFeature({
          title: '数据工厂',
          description: '编排数据处理任务，维护数据生产和加工流程。',
          capabilities: ['数据任务配置', '处理流程编排', '运行记录查看'],
        }),
      },
      {
        path: 'data-push',
        name: 'app-setting-data-push',
        ...settingFeature({
          title: '数据推送',
          description: '将应用数据按规则安全推送到外部目标。',
          capabilities: ['推送目标维护', '推送规则配置', '发送结果追踪'],
        }),
      },
      {
        path: 'process-analysis',
        name: 'app-setting-process-analysis',
        ...settingFeature({
          title: '流程分析',
          description: '分析应用内流程的执行效率、瓶颈和运行趋势。',
          capabilities: ['流程指标总览', '节点效率分析', '分析报告导出'],
        }),
      },
    ],
  },
];
export default appRoutes;
