import type { Component } from 'vue';
import type { DashboardWidgetType } from '~/types/dashboard';
import AppsWidget from './widgets/AppsWidget.vue';
import ChartsWidget from './widgets/ChartsWidget.vue';
import FavoritesWidget from './widgets/FavoritesWidget.vue';
import GreetingWidget from './widgets/GreetingWidget.vue';
import OnboardingWidget from './widgets/OnboardingWidget.vue';
import ShortcutWidget from './widgets/ShortcutWidget.vue';
import TodoWidget from './widgets/TodoWidget.vue';

/** 所有工作台卡片均在此显式注册，持久化布局只保存 type，不保存组件引用。 */
export const dashboardWidgetRegistry: Record<DashboardWidgetType, Component> = {
  onboarding: OnboardingWidget,
  greeting: GreetingWidget,
  shortcut: ShortcutWidget,
  todo: TodoWidget,
  favorites: FavoritesWidget,
  apps: AppsWidget,
  charts: ChartsWidget,
};
