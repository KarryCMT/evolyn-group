import { ContentTypeEnum, defHttp, http } from '@evolyn.do/utils';

/** 成员在当前租户中的生命周期状态；离职状态保留成员历史供组织页查询。 */
export type MemberStatus = 'active' | 'disabled' | 'resigned';

export interface MemberListQuery {
  departmentId?: number;
  status?: MemberStatus;
  keyword?: string;
  page?: number;
  pageSize?: number;
}

export interface MemberListItemDto {
  id: number;
  accountId: number;
  name: string;
  phone: string;
  email: string;
  avatar: string;
  status: MemberStatus;
  resignedAt: string | null;
  departments: { id: number; name: string }[];
  roles: { id: number; name: string }[];
}

export interface MemberPageDto {
  items: MemberListItemDto[];
  total: number;
}

/** 与通讯录批量导入模板一一对应的成员邀请档案。 */
export interface MemberInvitationPayload {
  name: string;
  identifier?: string;
  phone?: string;
  email?: string;
  departmentIds?: number[];
  departmentNames?: string[];
  alias?: string;
  employeeNo?: string;
  gender?: string;
  title?: string;
  employmentType?: string;
  hiredAt?: string;
  workLocation?: string;
  birthday?: string;
  education?: string;
}

export interface MemberInvitationDto extends MemberInvitationPayload {
  id: number;
  inviterMemberId: number;
  inviteToken: string;
  source: 'manual' | 'batch';
  status: 'pending' | 'accepted' | 'cancelled';
  profile: Omit<MemberInvitationPayload, 'name' | 'identifier' | 'phone' | 'email'>;
}

export interface MemberInvitationImportResult {
  successCount: number;
  failedRows: string[];
}

export interface PublicMemberInvitationLinkDto {
  id: number;
  token: string;
  enabled: boolean;
  creatorMemberId: number;
}

/** 按部门、状态、关键词分页查询当前租户成员。 */
export function listMembers(query: MemberListQuery = {}): Promise<MemberPageDto> {
  return http.get('/members', {
    departmentId: query.departmentId,
    status: query.status,
    keyword: query.keyword,
    page: query.page,
    pageSize: query.pageSize,
  });
}

/** 更新成员在当前租户内的状态；离职成员仍可在离职视图中查询。 */
export function updateMemberStatus(id: string, status: MemberStatus): Promise<null> {
  return http.put(`/members/${id}/status`, { status });
}

/** 整体替换成员的部门归属；空数组表示解除全部部门归属。 */
export function updateMemberDepartments(memberId: string, departmentIds: string[]): Promise<null> {
  return http.put(`/members/${memberId}/departments`, { departmentIds: departmentIds.map(Number) });
}

/** 保存手动填写的待接受成员邀请。 */
export function createMemberInvitation(
  payload: MemberInvitationPayload,
): Promise<MemberInvitationDto> {
  return http.post('/members/invitations', payload);
}

/** 上传符合通讯录模板的 Excel，返回逐行成功与失败结果。 */
export function importMemberInvitations(file: File): Promise<MemberInvitationImportResult> {
  const formData = new FormData();
  formData.append('file', file, file.name);
  return defHttp.post<MemberInvitationImportResult>({
    url: '/members/invitations/import',
    data: formData,
    headers: { 'Content-Type': ContentTypeEnum.FORM_DATA },
  });
}

/** 获取当前租户的公开邀请链接开关及 token。 */
export function getPublicMemberInvitationLink(): Promise<PublicMemberInvitationLinkDto> {
  return http.get('/members/invitation-link');
}

/** 开启或关闭公开邀请链接。 */
export function updatePublicMemberInvitationLink(
  enabled: boolean,
): Promise<PublicMemberInvitationLinkDto> {
  return http.put('/members/invitation-link', { enabled });
}
