export enum UploadFileStatus {
  pending = 'pending',
  uploading = 'uploading',
  done = 'done',
  error = 'error',
}

export interface IFile {
  // 上传状态
  status?: UploadFileStatus;
  base64?: string;
  //上传后的url
  url?: string;
  // 上传的文件存放路径
  path?: string;
  // 文件对象
  file: any;
  uid?: string;
  // 是否处于就绪状态
  isLoading?: boolean;
  // 初始化oss 为了能单独控制单文件
  client?: any;
  // 文件大小
  partSize?: number;
  // 上传进度
  percentage?: number;
  // 是否分片
  abortCheckpoint?: boolean;
  // 准备就绪状态
  isPlay?: boolean;
  // uploadId
  upload?: string;
  tempCheckpoint?: any;
  uploadName?: string;
  // 文件类型
  type?: string;
  // 文件大小
  size?: number;
  // 文件id
  fileId?: any;
  md5?: any;
  ossTag?: string;
  key?: string;
}
