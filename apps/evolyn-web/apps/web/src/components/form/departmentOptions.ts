import type { DepartmentDto } from '~/api/department';
import { getDepartmentTree } from '~/api/department';

/** Element Plus TreeSelect 需要的最小部门树投影。 */
export interface DepartmentOption {
  value: string;
  label: string;
  disabled: boolean;
  children?: DepartmentOption[];
}

const cachedDepartmentOptions = new Map<string, readonly DepartmentOption[]>();
const pendingDepartmentOptions = new Map<string, Promise<readonly DepartmentOption[]>>();

/**
 * 同一表单可能在子表单的多行中挂载很多部门选择器。部门树在当前租户会话内不随
 * 字段而变化，因此以模块级 Promise 合并并发请求，并缓存不可变投影供所有单元格复用。
 */
export function loadDepartmentOptions(
  tenantID: string | null,
): Promise<readonly DepartmentOption[]> {
  // 未获取到租户身份时不能复用缓存，避免会话切换期间跨租户展示组织数据。
  if (tenantID === null)
    return getDepartmentTree().then((departments) => departments.map(toDepartmentOption));
  const cached = cachedDepartmentOptions.get(tenantID);
  if (cached) return Promise.resolve(cached);
  const pending = pendingDepartmentOptions.get(tenantID);
  if (pending) return pending;
  const request = getDepartmentTree()
    .then((departments) => {
      const options = departments.map(toDepartmentOption);
      cachedDepartmentOptions.set(tenantID, options);
      return options;
    })
    .finally(() => {
      pendingDepartmentOptions.delete(tenantID);
    });
  pendingDepartmentOptions.set(tenantID, request);
  return request;
}

function toDepartmentOption(department: DepartmentDto): DepartmentOption {
  return {
    value: String(department.id),
    label: department.name,
    disabled: department.status !== 'active',
    children: department.children?.map(toDepartmentOption),
  };
}
