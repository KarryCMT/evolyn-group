import type { MobileAppItem, MobileAppMenu, MobileAppPage } from '~/types/app';
import { http } from '@evolyn.do/utils';

export function listApps(): Promise<MobileAppPage> {
  return http.get('/apps', { status: 'active', limit: 40 });
}

export function getAppByCode(code: string): Promise<MobileAppItem> {
  return http.get(`/apps/code/${encodeURIComponent(code)}`);
}

export function getAppMenuByCode(code: string): Promise<MobileAppMenu> {
  return http.get(`/apps/code/${encodeURIComponent(code)}/menu`);
}
