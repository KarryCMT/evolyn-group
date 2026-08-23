import type { RouteRecordRaw } from 'vue-router';

interface FormFeatureProps {
  title: string;
  description: string;
}

/** 非设计器类一级工作区使用独立视图，避免继续渲染字段面板和属性面板。 */
function workspaceFeature(props: FormFeatureProps) {
  return {
    component: () => import('~/pages/form/workspace-placeholder.vue'),
    props,
  };
}

function extensionFeature(props: FormFeatureProps) {
  return {
    component: () => import('~/pages/form/extensions/feature-placeholder.vue'),
    props,
  };
}

/**
 * 表单路由集合
 */
const formRoutes: RouteRecordRaw[] = [
  {
    path: '/app/:appCode/form/:formId',
    name: 'form',
    component: () => import('~/pages/form/index.vue'),
    meta: { public: false },
    // 一级工作区各自拥有布局；仅“表单设计”页加载三栏设计器。
    redirect: { name: 'form-design' },
    children: [
      {
        path: 'design',
        name: 'form-design',
        component: () => import('~/pages/form/design.vue'),
      },
      {
        path: 'workflow',
        name: 'form-workflow-design',
        ...workspaceFeature({
          title: '流程设计',
          description: '流程图、节点与流转规则将在工作流引擎接入后在此配置。',
        }),
      },
      {
        path: 'extensions',
        name: 'form-extensions',
        component: () => import('~/pages/form/extensions/index.vue'),
        redirect: { name: 'form-extension-collaboration' },
        children: [
          {
            path: 'data-collaboration',
            name: 'form-extension-collaboration',
            component: () => import('~/pages/form/extensions/data-collaboration.vue'),
          },
          {
            path: 'data-details',
            name: 'form-extension-details',
            ...extensionFeature({
              title: '数据详情',
              description: '配置数据详情页的呈现内容与查看行为。',
            }),
          },
          {
            path: 'notifications',
            name: 'form-extension-notifications',
            ...extensionFeature({
              title: '推送提醒',
              description: '设置表单数据变化后向成员发送的提醒。',
            }),
          },
          {
            path: 'submit-prompt',
            name: 'form-extension-submit-prompt',
            ...extensionFeature({
              title: '提交提示',
              description: '设置成员提交表单后看到的提示内容。',
            }),
          },
          {
            path: 'print',
            name: 'form-extension-print',
            ...extensionFeature({
              title: '打印模板',
              description: '维护表单数据的打印版式与字段映射。',
            }),
          },
          {
            path: 'ai',
            name: 'form-extension-ai',
            ...extensionFeature({
              title: '智能助手',
              description: '配置可用于该表单的智能能力与使用范围。',
            }),
          },
          {
            path: 'payment',
            name: 'form-extension-payment',
            ...extensionFeature({
              title: '在线支付',
              description: '设置支付字段、支付结果与订单数据的关联。',
            }),
          },
          {
            path: 'actions',
            name: 'form-extension-actions',
            ...extensionFeature({
              title: '自定义按钮',
              description: '为表单数据配置可执行的业务操作。',
            }),
          },
          {
            path: 'push',
            name: 'form-extension-push',
            ...extensionFeature({
              title: '数据推送',
              description: '按规则将表单数据推送至外部服务。',
            }),
          },
        ],
      },
      {
        path: 'publish',
        name: 'form-publish',
        ...workspaceFeature({
          title: '表单发布',
          description: '发布范围、填报入口与访问策略将在此统一配置。',
        }),
      },
      {
        path: 'data',
        name: 'form-data',
        ...workspaceFeature({
          title: '数据管理',
          description: '表单提交数据、筛选视图与批量操作将在此管理。',
        }),
      },
    ],
  },
];
export default formRoutes;
