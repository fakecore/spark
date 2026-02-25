import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import isToday from 'dayjs/plugin/isToday';
import isYesterday from 'dayjs/plugin/isYesterday';

dayjs.locale('zh-cn');
dayjs.extend(isToday);
dayjs.extend(isYesterday);

// Router 已迁移到 src/router.tsx，由 main.tsx 直接使用
export {};
