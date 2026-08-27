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
        component: () => import('~/pages/tenant/ProductCenterPage.vue'),
        meta: { title: '产品中心' },
      },
      {
        path: 'edition',
        name: 'tenant-edition',
        component: () => import('~/pages/tenant/edition.vue'),
        meta: { title: '版本信息' },
      },
      {
        path: 'profile',
        name: 'tenant-profile',
        component: () => import('~/pages/tenant/profile.vue'),
        meta: { title: '企业信息' },
      },
      {
        path: 'orders',
        name: 'tenant-orders',
        component: () => import('~/pages/tenant/orders.vue'),
        meta: { title: '订单信息' },
      },
      {
        path: 'organization',
        name: 'tenant-organization',
        component: () => import('~/pages/tenant/organization.vue'),
        meta: { title: '内部组织' },
      },
      {
        path: 'external-organization',
        name: 'tenant-external-organization',
        component: () => import('~/pages/tenant/external-organization.vue'),
        meta: { title: '互联组织' },
      },
      {
        path: 'member-fields',
        component: () => import('~/pages/tenant/member-fields.vue'),
        redirect: { name: 'tenant-member-fields' },
        children: [
          {
            path: 'fields',
            name: 'tenant-member-fields',
            component: () => import('~/components/tenant/memberFields/MemberFieldSettings.vue'),
            meta: { title: '成员字段设置' },
          },
          {
            path: 'cards',
            name: 'tenant-member-cards',
            component: () => import('~/components/tenant/memberFields/MemberCardDisplay.vue'),
            meta: { title: '成员卡片展示' },
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
            component: () => import('~/pages/tenant/SystemAdministratorsPage.vue'),
            meta: { title: '系统管理员' },
          },
          {
            path: 'apps',
            name: 'tenant-app-administrators',
            component: () => import('~/pages/tenant/ApplicationAdministratorsPage.vue'),
            meta: { title: '灵衍云管理员' },
          },
        ],
      },
      {
        path: 'permission-query',
        name: 'tenant-permission-query',
        component: () => import('~/pages/tenant/permission-query.vue'),
        meta: { title: '权限查询' },
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
        component: () => import('~/pages/tenant/UsageStatisticsPage.vue'),
        meta: { title: '使用统计' },
      },
      {
        path: 'enterprise-logs',
        name: 'tenant-enterprise-logs',
        component: () => import('~/pages/tenant/enterprise-logs.vue'),
        meta: { title: '企业日志' },
      },
      {
        path: 'product-logs',
        name: 'tenant-product-logs',
        component: () => import('~/pages/tenant/product-logs.vue'),
        meta: { title: '产品日志' },
      },
    ],
  },
];

export default tenantRoutes;
