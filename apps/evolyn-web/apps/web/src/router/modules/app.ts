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
        component: () => import('~/pages/app/setting/permissions.vue'),
      },
      {
        path: 'cross-app',
        name: 'app-setting-cross-app',
        component: () => import('~/pages/app/cross-app.vue'),
      },
      {
        path: 'basic',
        name: 'app-setting-basic',
        component: () => import('~/pages/app/setting/basic.vue'),
      },
      {
        path: 'management-groups',
        name: 'app-setting-management-groups',
        component: () => import('~/pages/app/setting/management-groups.vue'),
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
        component: () => import('~/pages/app/setting/data-push.vue'),
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
