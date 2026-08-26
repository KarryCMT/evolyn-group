import { http } from '@evolyn.do/utils';

/** 部门接口 DTO：与后端 iam/model.Department 和 DepartmentNode 对齐。 */
export interface DepartmentDto {
  id: number;
  parentId: number | null;
  name: string;
  order: number;
  status: 'active' | 'disabled';
  children?: DepartmentDto[];
}

export interface DepartmentPayload {
  name: string;
  parentId: number | null;
  order: number;
  status: 'active' | 'disabled';
}

/** 获取当前租户的完整部门树。 */
export function getDepartmentTree(): Promise<DepartmentDto[]> {
  return http.get('/departments/tree');
}

/** 创建部门；parentId 为 null 时创建一级部门。 */
export function createDepartment(payload: DepartmentPayload): Promise<DepartmentDto> {
  return http.post('/departments', payload);
}

/** 更新部门的名称、层级、排序及状态。 */
export function updateDepartment(id: number, payload: DepartmentPayload): Promise<DepartmentDto> {
  return http.put(`/departments/${id}`, payload);
}
