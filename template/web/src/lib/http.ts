import { getStorageItem, removeStorageItem, TOKEN_KEY } from '@/utils/storage';
import ky from 'ky';

export interface HttpResponse<T = unknown> {
  msg: string;
  code: number;
  data: T;
}

export interface PageResponse<T = unknown> {
  /** 页码 */
  current: number;
  /** 数据集合 */
  items: T[];
  /** 页宽 */
  pageSize: number;
  /** 总数 */
  total: number;
}

export type HR<T = unknown> = HttpResponse<T>;
export type PR<T = unknown> = HttpResponse<PageResponse<T>>;

export const http = ky.create({
  prefixUrl: '/api/v1',
  hooks: {
    beforeRequest: [
      (request) => {
        const token = getStorageItem(TOKEN_KEY);
        if (token) {
          request.headers.set('Authorization', `Bearer ${token}`);
        }
      },
    ],
    afterResponse: [
      (_request, _options, response) => {
        if (response.status === 401) {
          removeStorageItem(TOKEN_KEY);
          window.location.replace('/login');
        }
      },
    ],
  },
});

/**
 * 解析 JSON 响应并处理业务错误码
 */
export async function request<T = unknown>(responsePromise: Promise<Response>): Promise<HR<T>> {
  const response = await responsePromise;

  if (!response.ok) {
    throw new Error(response.statusText ?? '请求错误');
  }

  const json = (await response.json()) as HR<T>;

  if (json.code !== 200) {
    throw new Error(json.msg ?? '未知错误');
  }

  return json;
}
