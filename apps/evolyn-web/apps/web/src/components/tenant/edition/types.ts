import type { Component } from 'vue';
import type { EditionLimitSource, EditionMeteringStatus } from '~/types';

export type EditionTone = 'blue' | 'cyan' | 'green' | 'orange' | 'purple';

export interface EditionQuotaCard {
  /** 资源键（members/apps/storage_bytes/forms/workflow_runs_month） */
  id: string;
  icon: Component;
  title: string;
  tone: EditionTone;
  /** ready 展示真实用量；pending 展示「暂未启用统计」且不渲染进度条 */
  meteringStatus: EditionMeteringStatus;
  /** 0-100 使用率；pending 或不限量时为 0 且不渲染进度 */
  progress: number;
  usageLabel: string;
  limitLabel: string;
  note?: string;
  detail?: string;
  warning?: string;
  /** 上限解析来源与统计时间，详情弹窗展示 */
  limitSource?: EditionLimitSource;
  asOf?: string;
  /** 仅周期额度（如月度流程发起量）携带 */
  resetCycle?: string;
}

export interface EditionFeatureItem {
  available: boolean;
  icon: Component;
  id: string;
  meta?: string;
  title: string;
}

export interface EditionFeatureGroup {
  id: string;
  items: EditionFeatureItem[];
  requiresUpgrade?: boolean;
  title: string;
}
