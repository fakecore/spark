import Logo from '@/assets/logo.svg?react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { getUserProfile, logout } from '@/pages/login/service';
import { useUserStore } from '@/store/user';
import { removeStorageItem, TOKEN_KEY } from '@/utils/storage';
import { useQuery } from '@tanstack/react-query';
import { Outlet } from '@tanstack/react-router';
import { LogOut, Loader2 } from 'lucide-react';
import { useState } from 'react';

function Layout() {
  const setUserInfo = useUserStore((s) => s.setUserInfo);
  const [showLogoutDialog, setShowLogoutDialog] = useState(false);

  const { isLoading } = useQuery({
    queryKey: ['userProfile'],
    queryFn: async () => {
      const res = await getUserProfile();
      setUserInfo(res.data);
      return res.data;
    },
  });

  const handleLogout = async () => {
    try {
      await logout();
    } catch {
      // 即使 logout API 失败也清除本地状态
    } finally {
      removeStorageItem(TOKEN_KEY);
      window.location.href = '/';
    }
  };

  if (isLoading) {
    return (
      <div className="flex h-screen w-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="flex h-[100vh] flex-col">
      <div className="flex h-[48px] items-center justify-between bg-[#1A5AAF] px-[24px]">
        <div className="flex items-center gap-2">
          <Logo className="h-[28px] w-[28px]" />
          <div className="text-[14px] font-bold text-white">Project</div>
        </div>
        <div className="flex items-center gap-9 text-sm text-white">
          <LogOut onClick={() => setShowLogoutDialog(true)} className="cursor-pointer" size={18} />
        </div>
      </div>
      <div className="flex flex-1 overflow-y-auto bg-[#F5F5F5]">
        <Outlet />
      </div>

      <Dialog open={showLogoutDialog} onOpenChange={setShowLogoutDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认退出</DialogTitle>
            <DialogDescription>确认要退出登录吗？</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowLogoutDialog(false)}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleLogout}>
              确认退出
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default Layout;
