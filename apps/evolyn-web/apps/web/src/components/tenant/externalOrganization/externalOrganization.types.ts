/** 公开邀请链接的演示结构；后端接口就绪后可替换为真实 DTO。 */
export interface ExternalOrganizationLink {
  id: string;
  url: string;
  label: string;
  role: string;
  directoryPermission: string;
  enabled: boolean;
}

export type ExternalOrganizationInviteMode = 'public' | 'batch';
