import type { Component } from 'vue';

/** 管理后台侧栏中一个可访问的功能入口。 */
export interface TenantNavigationItem {
  key: string;
  label: string;
  path: string;
  /** 子路由仍高亮同一项，例如成员字段与管理员的页面内 Tab。 */
  activePath?: string;
  icon: Component;
}

/** 管理后台侧栏中的业务分组。 */
export interface TenantNavigationGroup {
  label: string;
  items: TenantNavigationItem[];
}

/** 页面内二级 Tab 的路由入口。 */
export interface TenantRouteTab {
  label: string;
  path: string;
}

/** 通用功能页在规划阶段展示的能力说明。 */
export interface TenantFeaturePageProps {
  title: string;
  description: string;
  capabilities: string[];
}
