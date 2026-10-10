import { describe, expect, it } from 'vitest';
import { createDefaultWorkbenchSchema } from '~/dashboard/defaultWorkbench';
import dashboardRoutes from '~/router/modules/dashboard';

describe('dashboard routes and workbench isolation', () => {
  it('registers independent design and revision preview routes', () => {
    const designRoute = dashboardRoutes.find(
      (route) => route.path === '/app/:appCode/dashboard/:dashboardCode/design',
    );
    expect(designRoute?.children?.find((route) => route.name === 'dashboard-design')?.path).toBe('');
    expect(dashboardRoutes.find((route) => route.name === 'dashboard-preview')?.path).toBe(
      '/app/:appCode/dashboard/:dashboardCode/preview',
    );
    expect(dashboardRoutes.find((route) => route.name === 'dashboard-extensions')?.path).toBe(
      '/app/:appCode/dashboard/:dashboardCode/extensions',
    );
    expect(designRoute?.children?.find((route) => route.name === 'dashboard-widget-edit')?.path).toBe(
      'widgets/:widgetId',
    );
  });

  it('keeps the enterprise workbench on the layout-only protocol', () => {
    const document = createDefaultWorkbenchSchema() as unknown as Record<string, unknown>;
    expect(document.version).toBe(1);
    expect(document.widgets).toBeInstanceOf(Array);
    expect(document).not.toHaveProperty('datasets');
    expect(document).not.toHaveProperty('publishScope');
  });
});
