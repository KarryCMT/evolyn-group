import type {
  MemberInvitationImportResult,
  MemberInvitationPayload,
  PublicMemberInvitationLinkDto,
} from '~/api/member';
import { computed, reactive, shallowRef } from 'vue';
import {
  createMemberInvitation,
  getPublicMemberInvitationLink,
  importMemberInvitations,
  updatePublicMemberInvitationLink,
} from '~/api/member';

export type MemberInvitationTab = 'manual' | 'batch' | 'public';

export interface MemberInvitationForm extends Required<
  Omit<MemberInvitationPayload, 'departmentIds' | 'departmentNames'>
> {
  departmentIds: number[];
}

function createManualForm(): MemberInvitationForm {
  return {
    name: '',
    identifier: '',
    phone: '',
    email: '',
    departmentIds: [],
    alias: '',
    employeeNo: '',
    gender: '',
    title: '',
    employmentType: '',
    hiredAt: '',
    workLocation: '',
    birthday: '',
    education: '',
  };
}

/** 邀请成员弹窗的请求状态与副作用边界，视图组件只接收状态和显式操作。 */
export function useMemberInvitation() {
  const activeTab = shallowRef<MemberInvitationTab>('manual');
  const submitting = shallowRef(false);
  const publicLinkLoading = shallowRef(false);
  const importResult = shallowRef<MemberInvitationImportResult | null>(null);
  const publicLink = shallowRef<PublicMemberInvitationLinkDto | null>(null);
  const form = reactive<MemberInvitationForm>(createManualForm());

  const publicInvitationUrl = computed(() => {
    if (!publicLink.value?.token || typeof window === 'undefined') return '';
    return `${window.location.origin}/auth/invite?token=${publicLink.value.token}`;
  });

  function clearManualForm() {
    Object.assign(form, createManualForm());
  }

  async function submitManualInvitation() {
    submitting.value = true;
    try {
      await createMemberInvitation({ ...form, departmentIds: [...form.departmentIds] });
    } finally {
      submitting.value = false;
    }
  }

  async function submitImport(file: File) {
    submitting.value = true;
    try {
      const result = await importMemberInvitations(file);
      importResult.value = result;
      return result;
    } finally {
      submitting.value = false;
    }
  }

  async function loadPublicLink() {
    if (publicLink.value || publicLinkLoading.value) return;
    publicLinkLoading.value = true;
    try {
      publicLink.value = await getPublicMemberInvitationLink();
    } finally {
      publicLinkLoading.value = false;
    }
  }

  async function setPublicLinkEnabled(enabled: boolean) {
    publicLinkLoading.value = true;
    try {
      publicLink.value = await updatePublicMemberInvitationLink(enabled);
    } finally {
      publicLinkLoading.value = false;
    }
  }

  function reset() {
    activeTab.value = 'manual';
    clearManualForm();
    importResult.value = null;
  }

  return {
    activeTab,
    submitting,
    publicLinkLoading,
    importResult,
    publicLink,
    form,
    publicInvitationUrl,
    clearManualForm,
    submitManualInvitation,
    submitImport,
    loadPublicLink,
    setPublicLinkEnabled,
    reset,
  };
}
