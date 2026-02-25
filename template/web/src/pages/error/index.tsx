import { Button } from '@/components/ui/button';
import { FileQuestion } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useNavigate } from '@tanstack/react-router';

const ErrorPage = () => {
  const navigate = useNavigate();
  const [count, setCount] = useState(5);

  useEffect(() => {
    if (count === 0) {
      navigate({ to: '/', replace: true });
      return;
    }
    const timer = setInterval(() => {
      setCount((prev) => prev - 1);
    }, 1000);
    return () => clearInterval(timer);
  }, [count, navigate]);

  return (
    <div className="flex h-full items-center justify-center bg-[#171717]">
      <FileQuestion className="h-[200px] w-[200px] text-muted-foreground/30" strokeWidth={1} />
      <div className="ml-[64px]">
        <div className="text-[28px] font-semibold leading-[40px] text-[#ddd]">发生错误 404</div>
        <div className="mt-3 text-[16px] text-[#ddd]">
          找不到请求的网址，系统将在
          <span className="mx-1 font-semibold text-[#1DF252]">{count}</span>
          秒内自动跳转回首页
        </div>
        <Button
          className="mt-[32px] flex h-[47px] w-[136px] items-center justify-center rounded-[8px] bg-[#0CD03D] text-[16px] font-medium text-white hover:bg-[#0CD03D]/90"
          onClick={() => navigate({ to: '/', replace: true })}
        >
          手动点击跳转
        </Button>
      </div>
      <div className="fixed bottom-4 left-0 flex w-full items-center justify-center">
        <div className="w-[238px] text-center text-[#8A8A8A]">Copyright © xxx</div>
      </div>
    </div>
  );
};

export default ErrorPage;
