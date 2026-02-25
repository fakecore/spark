import { postFileUploadUrl } from '@/request/service.ts';
import { IFile, UploadFileStatus } from '@/types/common';
import { filterSize } from '@/utils/utils.ts';
import OSS from 'ali-oss';

const config: any = {
  map_max_key: 0,
  checkpoints: {},
  parallel: 4,
  partSize: 1024 * 1024,
  fileMap: {},
  // bucket: 'zkrzpub',
  region: 'oss-cn-hangzhou',
};

const headers = {
  // 指定该Object被下载时的网页缓存行为。
  'Cache-Control': 'no-cache',
  // 指定该Object被下载时的内容编码格式。
  'Content-Encoding': 'utf-8',
  // 'x-oss-storage-class': 'Standard',
  // 指定过期时间，单位为毫秒。
  // 'x-oss-object-acl': 'private',
  // 'Content-Disposition': 'attachment',
  Expires: '1000',
  // 指定初始化分片上传时是否覆盖同名Object。此处设置为true，表示禁止覆盖同名Object。
  // 'x-oss-forbid-overwrite': 'true',
};

const handleChange = async (fileList: IFile[]) => {
  return Promise.all(
    fileList.map(async (item) => {
      item.client = null;
      item.isPlay = false;
      item.isLoading = false;
      item.abortCheckpoint = false;
      item.percentage = 0;
      item.status = UploadFileStatus.pending;
      item.type = item.file?.type;
      item.size = item.file?.size;
    }),
  );
};

export const aliOssUpload = async (
  biz_type: 1 | 2,
  fileList: IFile[],
  onProgress: (fileList: IFile[]) => void,
) => {
  // 初始化文件信息
  await handleChange(fileList);
  fileList.map(async (item) => {
    item.isLoading = true;
    try {
      const uploadInfo = await postFileUploadUrl({
        file_name: item.file.name,
        file_size: item.file.size.toString(),
        mime_type: item.file.type,
        biz_type,
      });
      // 直接返回上传url
      if (uploadInfo.data?.url?.url) {
        await fetch(uploadInfo.data?.url?.url, {
          method: 'PUT',
          body: item.file,
        });
        onProgress([
          {
            ...item,
            status: UploadFileStatus.done,
            percentage: 100,
            // TODO: 访问的文件地址，之后是否需要用私有桶
            url: uploadInfo.data.url.url.split('?')[0],
          },
        ]);
        return;
      }
      if (uploadInfo.data?.sts?.accessKeyId) {
        const {
          objectName,
          bucketName,
          accessKeyId,
          accessKeySecret,
          securityToken,
        } = uploadInfo.data!.sts;
        // const { ossPath, ossPrefix } = sts.data.bizFile;
        item.client = new OSS({
          // yourRegion填写Bucket所在地域。以华东1（杭州）为例，Region填写为oss-cn-hangzhou。
          region: config.region,
          // 填写Bucket名称。
          bucket: bucketName,
          secure: true,
          // 从STS服务获取的临时访问密钥（AccessKey ID和AccessKey Secret）。
          accessKeyId,
          accessKeySecret,
          // 从STS服务获取的安全令牌（SecurityToken）。
          stsToken: securityToken,
        });
        // item.ossTag = ossTag;
        item.url = `https://${bucketName}.oss-cn-hangzhou.aliyuncs.com/${objectName}`;
        item.path = `/${objectName}`;
        if ((item.file?.size ?? 0) < 1024) {
          item.client
            .put(item.path, item.file, {
              headers: {
                ...headers,
                // "x-oss-tagging": item.ossTag,
              },
            })
            .then(() => {
              item.percentage = 100;
              item.isLoading = false;
              onProgress([{ ...item, status: UploadFileStatus.done }]);
            });
        } else {
          await ossUpload(biz_type, item, fileList, onProgress);
        }
      }
    } catch (e) {
      console.log(e);
      onProgress([{ ...item, status: UploadFileStatus.error }]);
    }
  });
};

/**
 * @description 上传至OSS
 * @param {*} type 1公有仓库 2私有仓库
 * @param {*} item 文件信息
 * @param {*} fileList 所有文件
 * @param {*} onProgress 进度
 * @returns
 */
const ossUpload = async (
  type: 1 | 2,
  item: IFile,
  fileList: IFile[],
  onProgress: (fileList: IFile[]) => void,
) => {
  let isPass = {
    pass: true,
    filePath: '',
  };
  try {
    const { file, percentage, path } = item;
    item.partSize = 0;

    if ((percentage ?? 0) < 100 && file?.name.indexOf('.') !== -1) {
      item.status = UploadFileStatus.uploading;
      item.client
        .multipartUpload(path!, file, {
          parallel: config.parallel,
          partSize: config.partSize,
          headers: {
            ...headers,
          },
          mime: 'text/plain',
          progress: async (p: number, checkpoint: any, res: any) => {
            await onUploadProgress(item, p, checkpoint, res);
            onProgress(fileList);
          },
        })
        .then(async () => {
          if (item.tempCheckpoint?.uploadId) {
            delete config.checkpoints[item.tempCheckpoint.uploadId];
          }
          item.status = UploadFileStatus.done;
          onProgress(fileList);
        })
        .catch(async (err: any) => {
          console.log('err--', err);
          await resetUpload(err, type, item, fileList, onProgress);
        });
    }
  } catch (e: any) {
    //上传失败处理
    isPass = {
      ...e,
      pass: false,
      filePath: '',
    };
  }
  //上传成功返回filepath
  return isPass;
};

const change = (i: number, value: number) => {
  config.fileMap[i] = value;
  config.map_max_key = i;
};

const handle_network_speed_change = async (
  start_time: number,
  end_time: number,
  network_speed: number,
) => {
  // 如果超过10秒没有传输数据,则清空map
  if (start_time - config.map_max_key >= 10000) {
    config.fileMap = {};
  }
  for (let i = start_time; i <= end_time; i++) {
    const value = await config.fileMap[i];
    if (value) {
      await change(i, value + network_speed);
    } else {
      await change(i, network_speed);
    }
  }
};

/**
 * @description 获取上传的网络状态
 * @param {*} res 文件信息
 * @param {*} p 上传进度
 * @returns 网速度 network_speed
 */
const handle_network_speed = async (res: any, p: number) => {
  const spend_time = res.rt / 1000; //单位s
  const end_time = new Date(res.headers.date).getTime();
  const start_time = end_time - spend_time;
  let network_speed = parseInt((config.partSize / spend_time).toString()); // 每s中上传的字节(b)数
  if (p === 0) network_speed = 0;
  if (network_speed === 0) {
    // nothing to do
  } else {
    await handle_network_speed_change(start_time, end_time, network_speed);
  }
  return network_speed ? filterSize(network_speed) : 0;
};

const onUploadProgress = async (
  item: any,
  p: number,
  checkpoint: any,
  res: any,
) => {
  if (checkpoint) {
    config.checkpoints[checkpoint.uploadId] = checkpoint;
    item.speed = handle_network_speed(res, p);
    item.tempCheckpoint = checkpoint;
    item.abortCheckpoint = true;
    item.upload = checkpoint.uploadId;
  }
  // 改变上传状态
  item.isPlay = true;
  // 改变准备就绪状态
  if (item.isPlay) item.isLoading = false;
  // 上传进度
  item.percentage = Number((p * 100).toFixed(2));
  if (item.percentage === 100) {
    item.status = UploadFileStatus.done;
  }
};

const resetUpload = async (
  err: any,
  biz_type: 1 | 2,
  item: IFile,
  fileList: IFile[],
  onProgress: (fileList: IFile[]) => void,
) => {
  const msg = JSON.stringify(err);
  if (msg.indexOf('Error') !== -1) {
    if (item.client) {
      item.client.cancel();
    }
    const uploadInfo = await postFileUploadUrl({
      file_name: item.file.name,
      file_size: item.file.size.toString(),
      mime_type: item.file.type,
      biz_type,
    });
    const {
      objectName,
      bucketName,
      accessKeyId,
      accessKeySecret,
      securityToken,
    } = uploadInfo.data.sts!;
    item.client = new OSS({
      // yourRegion填写Bucket所在地域。以华东1（杭州）为例，Region填写为oss-cn-hangzhou。
      region: config.region,
      // 填写Bucket名称。
      bucket: bucketName,
      secure: true,
      // 从STS服务获取的临时访问密钥（AccessKey ID和AccessKey Secret）。
      accessKeyId,
      accessKeySecret,
      // 从STS服务获取的安全令牌（SecurityToken）。
      stsToken: securityToken,
      endpoint: 'oss-cn-hangzhou.aliyuncs.com',
    });
    // item.ossTag = ossTag;
    item.url = `https://${bucketName}.oss-cn-hangzhou.aliyuncs.com/${objectName}`;
    item.path = `/${objectName}`;
    await resumeMultipartUpload(item, fileList, onProgress);
  }
};

/**
 * @description 恢复上传
 */
const resumeMultipartUpload = async (
  item: IFile,
  fileList: IFile[],
  onProgress: (fileList: IFile[]) => void,
) => {
  // 恢复单文件
  if (item) {
    const { tempCheckpoint } = item;
    resumeUploadFile(item, tempCheckpoint, fileList, onProgress);
  } else {
    // 多文件
    Object.values(config.checkpoints).forEach((checkpoint: any) => {
      const { uploadId } = checkpoint;
      const index = fileList.findIndex((option) => option.upload === uploadId);
      const item = fileList[index];
      resumeUploadFile(item, checkpoint, fileList, onProgress);
    });
  }
};

/**
 * @description 恢复上传
 * @param {*} item 文件信息
 * @param {*} checkpoint 分片信息
 * @param {*} fileList 所选文件列表
 * @param {*} onProgress 进度
 */
const resumeUploadFile = (
  item: IFile,
  checkpoint: any,
  fileList: IFile[],
  onProgress: (fileList: IFile[]) => void,
) => {
  if (!checkpoint) return;
  const { uploadId, file } = checkpoint;
  try {
    const { percentage } = item;
    item.partSize = 0;
    item.status = UploadFileStatus.uploading;
    if ((percentage ?? 0) < 100 && file.name.indexOf('.') !== -1) {
      item.client
        .multipartUpload(uploadId, file, {
          parallel: config.parallel,
          partSize: config.partSize,
          headers,
          mime: 'text/plain',
          progress: async (p: number, checkpoint: any, res: any) => {
            await onUploadProgress(item, p, checkpoint, res);
          },
        })
        .then(async () => {
          if (checkpoint.uploadId) {
            delete config.checkpoints[checkpoint.uploadId];
          }
          item.status = UploadFileStatus.done;
          onProgress(fileList);
        })
        .catch(async (err: any) => {
          console.log('err--', err);
          // await resetUpload(err, item, fileList, onProgress);
        });
    }
  } catch {
    console.log('---err---');
  }
};
