import type { ComputedRef, InjectionKey, ShallowRef } from 'vue';
import type { DashboardWorkspaceTab } from '../workspace/dashboardWorkspace.types';
import type { useDashboardChartEditorSession } from '~/composables/useDashboardChartEditorSession';
import type { useDashboardDataCatalog } from '~/composables/useDashboardDataCatalog';
import type { useDashboardDesigner } from '~/composables/useDashboardDesigner';
import { inject } from 'vue';

export interface DashboardDesignWorkspaceContext {
  appCode: ComputedRef<string>;
  dashboardCode: ComputedRef<string>;
  designer: ReturnType<typeof useDashboardDesigner>;
  dataCatalog: ReturnType<typeof useDashboardDataCatalog>;
  chartEditor: ReturnType<typeof useDashboardChartEditorSession>;
  selectedDatasetId: ShallowRef<string | null>;
  renaming: ShallowRef<boolean>;
  saveDraft: () => Promise<void>;
  openPreview: () => Promise<void>;
  returnToApp: () => void;
  showHelp: () => void;
  navigateWorkspace: (tab: DashboardWorkspaceTab) => void;
  renameDashboard: (name: string, onSuccess: () => void) => Promise<void>;
  reloadConflict: () => Promise<void>;
}

export const dashboardDesignWorkspaceKey: InjectionKey<DashboardDesignWorkspaceContext> = Symbol(
  'dashboard-design-workspace',
);

export function useDashboardDesignWorkspaceContext(): DashboardDesignWorkspaceContext {
  const context = inject(dashboardDesignWorkspaceKey);
  if (!context) throw new Error('Dashboard design workspace is unavailable');
  return context;
}
