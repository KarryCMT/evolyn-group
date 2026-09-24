import type { IntelligentActionType } from '../schema';

export const ACTION_COLORS: Record<
  IntelligentActionType,
  { foreground: string; background: string }
> = {
  'create-record': { foreground: '#22a559', background: '#e9f8ef' },
  'update-record': { foreground: '#2f7cf6', background: '#eaf2ff' },
  'delete-record': { foreground: '#ef5b5b', background: '#fff0f0' },
  'send-notification': { foreground: '#8f5ce6', background: '#f3ecff' },
  'http-request': { foreground: '#b94fe2', background: '#faecff' },
  'condition': { foreground: '#e89a16', background: '#fff6df' },
  'data-transform': { foreground: '#00a99d', background: '#e7f8f6' },
};

export const NODE_SIZES = {
  trigger: { width: 286, height: 88 },
  action: { width: 286, height: 88 },
  end: { width: 122, height: 54 },
} as const;
