import type { RouteRecordRaw } from 'vue-router';
import type { TenantFeaturePageProps } from '~/types/tenant';

/** 规划阶段的功能页共用展示壳；实际业务落地时替换为各自领域页面。 */
function featurePage(props: TenantFeaturePageProps) {
  return {
    component: () => import('~/pages/tenant/feature.vue'),
    props,
    meta: { title: props.title },
  };
}

/** 租户管理后台路由集合，tenant/index.vue 只承担公共布局与子路由出口。 */
const tenantRoutes: RouteRecordRaw[] = [
  {
    path: '/tenant',
    name: 'tenant',
    component: () => import('~/pages/tenant/index.vue'),
    meta: { public: false, title: '企业管理' },
    redirect: { name: 'tenant-organization' },
    children: [
      {
        path: 'product-center',
        name: 'tenant-product-center',
        ...featurePage({
          title: '产品中心',
          description: '管理租户已开通的产品、模块与可用能力。',
          capabilities: ['产品与模块开通状态', '模块可见范围与成员授权入口', '产品使用概览'],
        }),
      },
      {
        path: 'edition',
        name: 'tenant-edition',
        ...featurePage({
          title: '版本信息',
          description: '集中查看当前套餐、配额与版本权益。',
          capabilities: ['套餐权益展示', '用量与配额预警', '版本升级指引'],
        }),
      },
      {
        path: 'profile',
        name: 'tenant-profile',
        ...featurePage({
          title: '企业信息',
          description: '维护企业主体资料和管理员可见的基础信息。',
          capabilities: ['企业资料维护', '企业标识与联系人', '变更记录'],
        }),
      },
      {
        path: 'orders',
        name: 'tenant-orders',
        ...featurePage({
          title: '订单信息',
          description: '查看订单、续费状态与历史交易记录。',
          capabilities: ['订单列表与详情', '续费与升级入口', '交易凭证下载'],
        }),
      },
      {
        path: 'organization',
        name: 'tenant-organization',
        ...featurePage({
          title: '内部组织',
          description: '维护部门、成员与角色，组织结构的变更将同步影响成员权限。',
          capabilities: ['部门树与部门成员', '成员邀请、筛选和导出', '角色分配与离职成员管理'],
        }),
      },
      {
        path: 'external-organization',
        name: 'tenant-external-organization',
        ...featurePage({
          title: '互联组织',
          description: '管理与外部组织协作时的成员、范围与访问边界。',
          capabilities: ['外部组织连接', '协作范围管理', '访问状态审计'],
        }),
      },
      {
        path: 'member-fields',
        component: () => import('~/pages/tenant/member-fields.vue'),
        redirect: { name: 'tenant-member-fields' },
        children: [
          {
            path: 'fields',
            name: 'tenant-member-fields',
            ...featurePage({
              title: '成员字段设置',
              description: '配置成员档案的系统字段、可见性和成员自助编辑权限。',
              capabilities: ['字段可见与可编辑权限', '字段类型与校验规则', '成员资料变更记录'],
            }),
          },
          {
            path: 'cards',
            name: 'tenant-member-cards',
            ...featurePage({
              title: '成员卡片展示',
              description: '选择成员卡片中可展示的字段，并分别预览桌面端和移动端。',
              capabilities: ['卡片字段排序', '桌面端与移动端预览', '成员信息脱敏策略'],
            }),
          },
        ],
      },
      {
        path: 'administrators',
        component: () => import('~/pages/tenant/administrators.vue'),
        redirect: { name: 'tenant-system-administrators' },
        children: [
          {
            path: 'system',
            name: 'tenant-system-administrators',
            ...featurePage({
              title: '系统管理员',
              description: '管理全局系统权限，建议仅授予必要的可信成员。',
              capabilities: ['管理员成员管理', '管理员角色范围', '授权与撤销审计'],
            }),
          },
          {
            path: 'apps',
            name: 'tenant-app-administrators',
            ...featurePage({
              title: '业务应用管理员',
              description: '为业务应用分配精细化的管理范围。',
              capabilities: ['应用管理员分组', '应用范围授权', '管理员操作记录'],
            }),
          },
        ],
      },
      {
        path: 'permission-query',
        name: 'tenant-permission-query',
        ...featurePage({
          title: '权限查询',
          description: '从成员、角色和应用维度追溯当前实际权限。',
          capabilities: ['成员权限视图', '角色授权关系', '权限来源追溯'],
        }),
      },
      {
        path: 'enterprise-settings',
        name: 'tenant-enterprise-settings',
        ...featurePage({
          title: '企业设置',
          description: '配置企业安全、协作策略和基础运行参数。',
          capabilities: ['企业安全与单点登录', '企业文化与品牌展示', '协作与 AI 能力开关'],
        }),
      },
      {
        path: 'product-settings',
        name: 'tenant-product-settings',
        ...featurePage({
          title: '产品设置',
          description: '配置产品协作方式、默认工作台与外部服务集成。',
          capabilities: ['应用分享与管理组', '产品样式与工作台', '支付、知识库等集成开关'],
        }),
      },
      {
        path: 'message-settings',
        name: 'tenant-message-settings',
        ...featurePage({
          title: '消息推送',
          description: '统一维护租户级消息通知渠道与推送策略。',
          capabilities: ['通知渠道配置', '事件订阅管理', '消息发送记录'],
        }),
      },
      {
        path: 'usage',
        name: 'tenant-usage',
        ...featurePage({
          title: '使用统计',
          description: '从成员、产品与时间范围分析租户使用情况。',
          capabilities: ['访问和活跃趋势', '功能使用分布', '统计报表导出'],
        }),
      },
      {
        path: 'enterprise-logs',
        name: 'tenant-enterprise-logs',
        ...featurePage({
          title: '企业日志',
          description: '审计企业管理与关键配置的操作记录。',
          capabilities: ['按操作者与时间筛选', '操作详情查看', '日志导出与留存策略'],
        }),
      },
      {
        path: 'product-logs',
        name: 'tenant-product-logs',
        ...featurePage({
          title: '产品日志',
          description: '查看产品模块内的重要事件和运行记录。',
          capabilities: ['产品事件查询', '异常事件筛选', '日志导出'],
        }),
      },
    ],
  },
];

export default tenantRoutes;
