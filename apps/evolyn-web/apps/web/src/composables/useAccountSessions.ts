import { shallowRef } from 'vue';
import { listMySessions, revokeMySession } from '~/api/account';
import type { AccountSession } from '~/types';

/** 账号设备会话的加载与撤销状态收口。 */
export function useAccountSessions() {
  const sessions = shallowRef<AccountSession[]>([]);
  const loading = shallowRef(false);
  const revokingSID = shallowRef('');

  async function loadSessions() {
    if (loading.value) return;
    loading.value = true;
    try {
      sessions.value = await listMySessions();
    } finally {
      loading.value = false;
    }
  }

  async function revokeSession(sid: string) {
    revokingSID.value = sid;
    try {
      await revokeMySession(sid);
      // 成功后本地移除，避免等待下一次打开抽屉才能反映设备已下线。
      sessions.value = sessions.value.filter((session) => session.sid !== sid);
    } finally {
      revokingSID.value = '';
    }
  }

  return {
    sessions,
    loading,
    revokingSID,
    loadSessions,
    revokeSession,
  };
}
