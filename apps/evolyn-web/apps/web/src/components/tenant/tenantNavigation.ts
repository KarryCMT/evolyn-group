import {
  RiApps2Fill,
  RiBarChartBoxFill,
  RiContactsBook3Fill,
  RiFileList3Fill,
  RiHomeGearFill,
  RiNotification3Fill,
  RiSettings3Fill,
} from '@remixicon/vue';
import type { TenantNavigationGroup } from '~/types/tenant';

/**
 * 租户后台的菜单与路由入口保持在同一个配置中，避免菜单名称和路径分别维护后漂移。
 * 需要新增后台模块时，先在 tenant.ts 注册路由，再补充这里的可见入口。
 */
export const tenantNavigationGroups: TenantNavigationGroup[] = [
  {
    label: '基本信息',
    items: [
      {
        key: 'product-center',
        label: '产品中心',
        path: '/tenant/product-center',
        icon: RiApps2Fill,
      },
      { key: 'edition', label: '版本信息', path: '/tenant/edition', icon: RiBarChartBoxFill },
      { key: 'profile', label: '企业信息', path: '/tenant/profile', icon: RiHomeGearFill },
      { key: 'orders', label: '订单信息', path: '/tenant/orders', icon: RiFileList3Fill },
    ],
  },
  {
    label: '通讯录',
    items: [
      {
        key: 'organization',
        label: '内部组织',
        path: '/tenant/organization',
        icon: RiContactsBook3Fill,
      },
      {
        key: 'external-organization',
        label: '互联组织',
        path: '/tenant/external-organization',
        icon: RiContactsBook3Fill,
      },
      {
        key: 'member-fields',
        label: '成员信息管理',
        path: '/tenant/member-fields/fields',
        activePath: '/tenant/member-fields',
        icon: RiContactsBook3Fill,
      },
    ],
  },
  {
    label: '权限中心',
    items: [
      {
        key: 'administrators',
        label: '管理员',
        path: '/tenant/administrators/system',
        activePath: '/tenant/administrators',
        icon: RiSettings3Fill,
      },
      {
        key: 'permission-query',
        label: '权限查询',
        path: '/tenant/permission-query',
        icon: RiSettings3Fill,
      },
    ],
  },
  {
    label: '管理工具',
    items: [
      {
        key: 'enterprise-settings',
        label: '企业设置',
        path: '/tenant/enterprise-settings',
        icon: RiHomeGearFill,
      },
      {
        key: 'product-settings',
        label: '产品设置',
        path: '/tenant/product-settings',
        icon: RiApps2Fill,
      },
      {
        key: 'message-settings',
        label: '消息推送',
        path: '/tenant/message-settings',
        icon: RiNotification3Fill,
      },
      { key: 'usage', label: '使用统计', path: '/tenant/usage', icon: RiBarChartBoxFill },
    ],
  },
  {
    label: '日志审计',
    items: [
      {
        key: 'enterprise-logs',
        label: '企业日志',
        path: '/tenant/enterprise-logs',
        icon: RiFileList3Fill,
      },
      {
        key: 'product-logs',
        label: '产品日志',
        path: '/tenant/product-logs',
        icon: RiFileList3Fill,
      },
    ],
  },
];
