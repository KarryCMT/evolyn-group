// evolyn 统一请求层的项目适配点：本文件对齐后端 httpx 统一响应 { code, errCode, msg, data }
// 与 ADR-008 业务错误约定，其余文件保持通用不必动。
// 关键行为：
// - 成功响应解包返回 data 字段（空数据规范化为 null）
// - 失败（非 2xx / 网络 / 超时）统一抛 ApiError（status + errCode，msg 可直接展示）
// - 令牌经 ../auth 读取，Authorization 头固定 Bearer 方案
// - 接口地址惰性读取 useGlobSetting（配置注入发生在应用入口，晚于本模块加载）

import type { AxiosInstance, AxiosResponse } from 'axios';
import { clone } from 'lodash-es';
import type { RequestOptions, Result } from '../types/axios';
import type { AxiosTransform, CreateAxiosOptions } from './axiosTransform';
import { VAxios } from './Axios';
import { ApiError } from './error';
import { getRequestMessage } from './message';
import { useGlobSetting } from '../setting/globSetting';
import { RequestEnum, ContentTypeEnum } from '../enums/httpEnum';
import { isString } from '../is';
import { getToken } from '../auth';
import { setObjToUrlParams, deepMerge } from '../is';
import { joinTimestamp, formatRequestDate } from './helper';
import { AxiosRetry } from './axiosRetry';
import axios from 'axios';

/**
 * @description: 数据处理（成功/失败两条路径的归一）
 */
const transform: AxiosTransform = {
  /**
   * @description: 成功响应处理。HTTP 2xx 即成功（后端语义错误走非 2xx + errCode），
   * 默认解包统一响应返回 data 字段。
   */
  transformResponseHook: (res: AxiosResponse<Result>, options: RequestOptions) => {
    const { isTransformResponse, isReturnNativeResponse } = options;
    // 是否返回原生响应（需要读取响应头等场景）
    if (isReturnNativeResponse) {
      return res;
    }
    // 不解包，直接返回整个统一响应体（页面需要 code/errCode/msg 时开启）
    if (!isTransformResponse) {
      return res.data;
    }
    // 默认：返回 data 字段，空数据规范化为 null
    return res.data?.data ?? null;
  },

  // 请求之前处理 config
  beforeRequestHook: (config, options) => {
    // 配置惰性兜底：全局配置由应用入口注入（晚于本模块加载），不能在模块顶层快照
    const glob = useGlobSetting();
    const apiUrl = options.apiUrl ?? glob.apiUrl;
    const urlPrefix = options.urlPrefix ?? glob.urlPrefix;
    const { joinPrefix, joinParamsToUrl, formatDate, joinTime = true } = options;

    // urlPrefix 为空串/未配置时不拼接，避免出现 "undefined/api/..."
    if (joinPrefix && urlPrefix) {
      config.url = `${urlPrefix}${config.url}`;
    }

    if (apiUrl && isString(apiUrl) && !/https?:\/\//.test(config.url || '')) {
      config.url = `${apiUrl}${config.url}`;
    }
    const params = config.params || config.data || {};
    const data = config.data || false;
    formatDate && data && !isString(data) && formatRequestDate(data);
    if (config.method?.toUpperCase() === RequestEnum.GET) {
      if (!isString(params)) {
        // 给 get 请求加上时间戳参数，避免从缓存中拿数据。
        config.params = Object.assign(params || {}, joinTimestamp(joinTime, false));
      } else {
        // 兼容restful风格
        config.url = config.url + params + `${joinTimestamp(joinTime, true)}`;
        config.params = undefined;
      }
    } else {
      if (!isString(params)) {
        formatDate && formatRequestDate(params);
        if (
          Reflect.has(config, 'data') &&
          config.data &&
          (Object.keys(config.data).length > 0 || config.data instanceof FormData)
        ) {
          config.data = data;
          config.params = undefined;
        } else {
          // 非GET请求如果没有提供data，则将params视为data
          config.data = params;
          config.params = undefined;
        }
        if (joinParamsToUrl) {
          config.url = setObjToUrlParams(
            config.url as string,
            Object.assign({}, config.params, config.data),
          );
        }
      } else {
        // 兼容restful风格
        config.url = config.url + params;
        config.params = undefined;
      }
    }
    return config;
  },

  /**
   * @description: 请求拦截器：携带 Bearer 令牌（每次请求动态读取，登录后立即生效）
   */
  requestInterceptors: (config, options) => {
    const token = getToken();
    if (token && (config as Recordable)?.requestOptions?.withToken !== false) {
      (config as Recordable).headers.Authorization = options.authenticationScheme
        ? `${options.authenticationScheme} ${token}`
        : token;
    }
    return config;
  },

  /**
   * @description: 响应拦截器：透传
   */
  responseInterceptors: (res: AxiosResponse<any>) => {
    return res;
  },

  /**
   * @description: 响应错误处理：统一归一为 ApiError。
   * 错误文案以后端 envelope.msg 为权威，缺失（网关错误页等非 JSON）时按状态码兜底；
   * errorMessageMode 为 none（默认）时不弹提示，由调用方按 errCode 自行分支处理。
   */
  responseInterceptorsCatch: (axiosInstance: AxiosInstance, error: any) => {
    if (axios.isCancel(error)) {
      return Promise.reject(error);
    }

    const { response, config } = error || {};
    const { errorMessageMode } = config?.requestOptions || {};

    // 请求未到达服务端（断网/超时/中断）：统一网络异常语义，status=0
    if (!response) {
      return Promise.reject(new ApiError('网络异常，请检查网络后重试', 0));
    }

    // 统一响应体：非 JSON（如网关错误页）时为 undefined，走兜底文案
    const envelope = response.data as Result | undefined;
    const apiError = new ApiError(
      envelope?.msg || `请求失败（HTTP ${response.status}）`,
      response.status,
      envelope?.errCode,
    );

    // errorMessageMode='modal' 显示重要错误弹窗；'message' 轻量提示；默认 'none' 静默
    const { createMessage, createErrorModal } = getRequestMessage();
    if (errorMessageMode === 'modal') {
      createErrorModal({ title: '错误提示', content: apiError.message });
    } else if (errorMessageMode === 'message') {
      createMessage.error(apiError.message);
    }

    // GET 请求自动重试（默认关闭，需在 requestOptions.retryRequest 开启）
    const { isOpenRetry } = config?.requestOptions?.retryRequest || {};
    if (config?.method?.toUpperCase() === RequestEnum.GET && isOpenRetry) {
      // @ts-ignore error 此时必为 AxiosError（axios 网络层错误）
      new AxiosRetry().retry(axiosInstance, error);
    }

    return Promise.reject(apiError);
  },
};

function createAxios(opt?: Partial<CreateAxiosOptions>) {
  return new VAxios(
    // 深度合并
    deepMerge(
      {
        // See https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication#authentication_schemes
        // JWT Bearer 方案，与后端认证中间件约定一致
        authenticationScheme: 'Bearer',
        timeout: 1000 * 1000,

        headers: { 'Content-Type': ContentTypeEnum.JSON },
        // 如果是form-data格式
        // headers: { 'Content-Type': ContentTypeEnum.FORM_URLENCODED },
        // 数据处理方式
        transform: clone(transform),
        // 配置项，下面的选项都可以在独立的接口请求中覆盖
        requestOptions: {
          // 不拼接 urlPrefix：接口地址统一由 apiUrl（全局配置注入）承担
          joinPrefix: false,
          // 是否返回原生响应头 比如：需要获取响应头时使用该属性
          isReturnNativeResponse: false,
          // 默认解包统一响应返回 data 字段
          isTransformResponse: true,
          // post请求的时候添加参数到url
          joinParamsToUrl: false,
          // 格式化提交参数时间
          formatDate: true,
          // 错误提示默认静默：由调用方按 errCode 分支处理（应用可在单请求覆盖）
          errorMessageMode: 'none',
          // 接口地址不设默认值，由 beforeRequestHook 惰性读全局配置
          // 是否加入时间戳
          joinTime: true,
          // 忽略重复请求取消
          ignoreCancelToken: true,
          // 是否携带token
          withToken: true,
          // 是否加密
          useCipher: false,
          retryRequest: {
            isOpenRetry: false,
            count: 5,
            waitTime: 100,
          },
        },
      },
      opt || {},
    ),
  );
}

export const defHttp = createAxios();

// other api url
// export const otherHttp = createAxios({
//   requestOptions: {
//     apiUrl: 'xxx',
//     urlPrefix: 'xxx',
//   },
// });
