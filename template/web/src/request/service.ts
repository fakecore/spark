import { http, request } from '@/lib/http';

export interface FileUploadUrlReplySts {
  accessKeyId: string;
  accessKeySecret: string;
  bucketName: string;
  expiration: string;
  objectName: string;
  region: string;
  securityToken: string;
}

export interface FileUploadUrlReplyUrl {
  expiration: string;
  method: string;
  signedHeaders: { [key: string]: string };
  url: string;
}

export interface FileUploadRes {
  sts?: FileUploadUrlReplySts;
  url?: FileUploadUrlReplyUrl;
  [property: string]: unknown;
}

export function postFileUploadUrl(params: {
  file_name?: string;
  file_size?: string;
  mime_type?: string;
  biz_type?: 1 | 2;
}) {
  return request<FileUploadRes>(http.post('file/upload-url', { json: params }));
}
