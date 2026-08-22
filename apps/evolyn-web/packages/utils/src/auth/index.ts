import { CacheTypeEnum } from '../enums/cacheEnum';
import projectSetting from '../setting';
import { TOKEN_KEY } from '../enums/cacheEnum';

const { permissionCacheType } = projectSetting;
const isLocal = permissionCacheType === CacheTypeEnum.LOCAL;

export function getToken() {
  return window.localStorage.getItem(TOKEN_KEY);
}

export function setToken(info: string) {
  if (isLocal) {
    window.localStorage.setItem(TOKEN_KEY, info);
  } else {
    window.sessionStorage.setItem(TOKEN_KEY, info);
  }
}

export function removeToken() {
  if (isLocal) {
    window.localStorage.removeItem(TOKEN_KEY);
  } else {
    window.sessionStorage.removeItem(TOKEN_KEY);
  }
}
