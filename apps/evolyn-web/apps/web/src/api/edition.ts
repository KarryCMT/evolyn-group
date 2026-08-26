// 版本信息域接口：与后端 /api/v1/editions* 一一对应
// （见 evolyn-core internal/platform/edition/controller/edition.go）
import { http } from '@evolyn.do/utils';
import type { CurrentEdition } from '~/types';

/**
 * 当前租户版本信息概览（GET /editions/current）：订阅（到期即时投影）、
 * 资源容量（meteringStatus=pending 的资源无已用值）与功能权益；
 * 需要租户管理员基线权限 editions:get，普通成员返回 FORBIDDEN
 */
export function getCurrentEdition(): Promise<CurrentEdition> {
  return http.get('/editions/current');
}
