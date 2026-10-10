import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiError } from '../error';
import { ERROR_CODES, SESSION_INVALIDATING_ERROR_CODES } from '../errorCodes';
import { request, setUnauthorizedHandler } from '../http';
import { defHttp } from '../instance';

describe('统一请求层的 401 分流', () => {
  const unauthorizedHandler = vi.fn();

  beforeEach(() => {
    vi.restoreAllMocks();
    unauthorizedHandler.mockReset();
    setUnauthorizedHandler(unauthorizedHandler);
  });

  it('保留业务型 401 的后端文案和错误码', async () => {
    const error = new ApiError('该手机号未注册', 401, ERROR_CODES.AUTH_ACCOUNT_NOT_FOUND);
    vi.spyOn(defHttp, 'request').mockRejectedValueOnce(error);

    await expect(request('/auth/sms/send', { method: 'POST' })).rejects.toBe(error);
    expect(unauthorizedHandler).not.toHaveBeenCalled();
  });

  it.each(SESSION_INVALIDATING_ERROR_CODES)('对会话失效错误码 %s 触发全局处理', async (errCode) => {
    const error = new ApiError('会话不可用', 401, errCode);
    vi.spyOn(defHttp, 'request').mockRejectedValueOnce(error);

    await expect(request('/auth/token/switch', { method: 'POST' })).rejects.toBe(error);
    expect(unauthorizedHandler).toHaveBeenCalledOnce();
    expect(unauthorizedHandler).toHaveBeenCalledWith(error);
  });

  it('未知 401 不臆断为会话失效', async () => {
    const error = new ApiError('上游认证失败', 401);
    vi.spyOn(defHttp, 'request').mockRejectedValueOnce(error);

    await expect(request('/gateway/resource')).rejects.toBe(error);
    expect(unauthorizedHandler).not.toHaveBeenCalled();
  });

  it('允许调用方显式跳过会话失效处理', async () => {
    const error = new ApiError('请先登录', 401, ERROR_CODES.UNAUTHORIZED);
    vi.spyOn(defHttp, 'request').mockRejectedValueOnce(error);

    await expect(request('/auth/userinfo', { skipUnauthorizedHandler: true })).rejects.toBe(error);
    expect(unauthorizedHandler).not.toHaveBeenCalled();
  });
});
