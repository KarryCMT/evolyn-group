import type { AppConf } from '~/types/auth';
import { http } from '@evolyn.do/utils';
import { JSEncrypt } from 'jsencrypt';

let confPromise: Promise<AppConf> | null = null;

function getAppConf(): Promise<AppConf> {
  if (!confPromise) {
    confPromise = http.get<AppConf>('/app/conf').catch((error: unknown) => {
      confPromise = null;
      throw error;
    });
  }
  return confPromise;
}

/** 密码只以平台公钥加密后的密文传输。 */
export async function encryptPassword(plain: string): Promise<string> {
  const conf = await getAppConf();
  if (conf.pki.algorithm !== 'rsa' || !conf.pki.keys.public_key) {
    throw new Error('密码加密公钥不可用，请刷新后重试');
  }

  const encryptor = new JSEncrypt();
  encryptor.setPublicKey(conf.pki.keys.public_key);
  const cipher = encryptor.encrypt(plain);
  if (cipher === false) throw new Error('密码加密失败，请刷新后重试');
  return cipher;
}
