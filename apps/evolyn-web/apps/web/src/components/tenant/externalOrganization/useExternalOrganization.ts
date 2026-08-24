import { shallowRef } from 'vue';
import type {
  ExternalOrganizationInviteMode,
  ExternalOrganizationLink,
} from './externalOrganization.types';

const defaultLink: ExternalOrganizationLink = {
  id: 'link-1',
  url: 'https://www.lingyanyun.com/portal/corp_connect/7Gd8a...',
  label: '',
  role: '',
  directoryPermission: '代管通讯录',
  enabled: true,
};

/** 集中管理页面内的演示状态，保证卡片、编辑弹窗和邀请面板同步更新。 */
export function useExternalOrganization() {
  const links = shallowRef<ExternalOrganizationLink[]>([]);
  const inviteVisible = shallowRef(false);
  const activeInviteMode = shallowRef<ExternalOrganizationInviteMode>('public');
  const editingLink = shallowRef<ExternalOrganizationLink | null>(null);

  function openInvite() {
    activeInviteMode.value = 'public';
    inviteVisible.value = true;
  }

  function addLink() {
    const serial = links.value.length + 1;
    links.value = [
      ...links.value,
      {
        ...defaultLink,
        id: `link-${serial}`,
        url: defaultLink.url.replace('7Gd8a', `${serial}Gd8a`),
      },
    ];
  }

  function updateLink(link: ExternalOrganizationLink) {
    links.value = links.value.map((item) => (item.id === link.id ? link : item));
  }

  function removeLink(id: string) {
    links.value = links.value.filter((link) => link.id !== id);
  }

  return {
    links,
    inviteVisible,
    activeInviteMode,
    editingLink,
    openInvite,
    addLink,
    updateLink,
    removeLink,
  };
}
