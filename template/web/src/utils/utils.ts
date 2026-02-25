const _baseUrl: string = process.env.VITE_PUBLIC_URL || '/';
export const baseUrl: string = _baseUrl?.endsWith('/')
  ? _baseUrl.slice(0, -1)
  : _baseUrl;

// 手机号校验
export const isMobile = (mobile: string) => {
  const reg = /^1[2-9]\d{9}$/;
  return reg.test(mobile);
};

export const formatDuration = (seconds: number) => {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  if (hours > 0) {
    return `${hours}小时${minutes}分${secs}秒`;
  }

  if (minutes > 0) {
    return `${minutes}分${secs}秒`;
  }
  return `${secs}秒`;
};

export const formatGameplayEndState = (state: number): string => {
  const StateMap: { [key: number]: string } = {
    0: '正常通关',
    1: '主动结束',
    2: '异常结束',
  };
  return StateMap[state] || '未知状态';
};

const pow1024 = (num: number) => {
  return Math.pow(1024, num);
};

export const filterSize = (size: number) => {
  if (!size) return '-';
  if (size < pow1024(1)) return size + ' B';
  if (size < pow1024(2)) return (size / pow1024(1)).toFixed(2) + ' KB';
  if (size < pow1024(3)) return (size / pow1024(2)).toFixed(2) + ' MB';
  if (size < pow1024(4)) return (size / pow1024(3)).toFixed(2) + ' GB';
  return (size / pow1024(4)).toFixed(2) + ' TB';
};
