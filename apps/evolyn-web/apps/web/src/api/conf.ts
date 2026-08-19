// 应用配置接口：GET /app/conf（匿名公开，见 evolyn-core internal/platform/controller/conf.go）。
// 提供带缓存的单例获取（区号/能力开关/RSA 公钥每次页面加载只需拉一次）与
// 密码加密助手（登录/改密的密码字段必须加密上送，明文不经过传输层）
import { JSEncrypt } from 'jsencrypt'
import { http } from './http'
import type { AppConf } from '~/types'

/** 应用配置缓存：并发调用共享同一 Promise，失败后允许重试 */
let confPromise: Promise<AppConf> | null = null

/** 获取应用配置（含 RSA 公钥）；失败清缓存，下次调用重新拉取 */
export function getAppConf(): Promise<AppConf> {
  if (!confPromise) {
    confPromise = http.get<AppConf>('/app/conf').catch((err: unknown) => {
      confPromise = null
      throw err
    })
  }
  return confPromise
}

/**
 * 以 /app/conf 下发的 RSA 公钥加密密码明文（服务端持私钥解密后再 bcrypt 校验）。
 * 公钥未就绪（配置拉取失败/算法不支持）时抛错，由调用方提示用户重试
 */
export async function encryptPassword(plain: string): Promise<string> {
  const conf = await getAppConf()
  if (conf.pki.algorithm !== 'rsa' || !conf.pki.keys.public_key) {
    throw new Error('密码加密公钥不可用，请刷新页面重试')
  }

  const encryptor = new JSEncrypt()
  encryptor.setPublicKey(conf.pki.keys.public_key)
  const cipher = encryptor.encrypt(plain)
  if (cipher === false) {
    throw new Error('密码加密失败，请刷新页面重试')
  }
  return cipher
}
