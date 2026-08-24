import type { Component } from 'vue';

export type EditionTone = 'blue' | 'cyan' | 'green' | 'orange' | 'purple';

export interface EditionQuotaCard {
  detail?: string;
  icon: Component;
  id: string;
  limitLabel: string;
  note?: string;
  progress: number;
  title: string;
  tone: EditionTone;
  usageLabel: string;
  warning?: string;
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
